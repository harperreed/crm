// ABOUTME: CLI commands for managing CRM contacts.
// ABOUTME: Provides add, list, show, edit, and remove subcommands under "contact".

package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/google/uuid"
	"github.com/harperreed/crm/internal/models"
	"github.com/harperreed/crm/internal/storage"
	"github.com/spf13/cobra"
)

var contactCmd = &cobra.Command{
	Use:     "contact",
	Aliases: []string{"c"},
	Short:   "Manage contacts",
}

var contactAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new contact",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c := models.NewContact(args[0])

		email, _ := cmd.Flags().GetString("email")
		phone, _ := cmd.Flags().GetString("phone")
		fields, _ := cmd.Flags().GetStringArray("field")
		tags, _ := cmd.Flags().GetStringSlice("tag")

		c.Email = email
		c.Phone = phone

		for _, f := range fields {
			k, v, ok := strings.Cut(f, "=")
			if !ok {
				return fmt.Errorf("invalid field format %q, expected KEY=VALUE", f)
			}
			c.Fields[k] = v
		}
		c.Tags = tags

		if err := store.CreateContact(c); err != nil {
			return err
		}

		cyan := color.New(color.FgCyan)
		out("Created contact %s\n", cyan.Sprint(c.ID))
		return nil
	},
}

var contactListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls", "l"},
	Short:   "List contacts",
	RunE: func(cmd *cobra.Command, args []string) error {
		tag, _ := cmd.Flags().GetString("tag")
		search, _ := cmd.Flags().GetString("search")
		limit, _ := cmd.Flags().GetInt("limit")

		filter := &storage.ContactFilter{
			Search: search,
			Limit:  limit,
		}
		if tag != "" {
			filter.Tag = &tag
		}

		contacts, err := store.ListContacts(filter)
		if err != nil {
			return err
		}

		if len(contacts) == 0 {
			outln("No contacts found.")
			return nil
		}

		cyan := color.New(color.FgCyan)
		bold := color.New(color.Bold)
		for _, c := range contacts {
			out("%s  %s", cyan.Sprint(c.ID), bold.Sprint(c.Name))
			if c.Email != "" {
				out("  <%s>", c.Email)
			}
			if len(c.Tags) > 0 {
				out("  [%s]", strings.Join(c.Tags, ", "))
			}
			outln()
		}
		return nil
	},
}

var contactShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show contact details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := resolveContact(args[0])
		if err != nil {
			return err
		}

		cyan := color.New(color.FgCyan)
		bold := color.New(color.Bold)

		out("ID:      %s\n", cyan.Sprint(c.ID))
		out("Name:    %s\n", bold.Sprint(c.Name))
		if c.Email != "" {
			out("Email:   <%s>\n", c.Email)
		}
		if c.Phone != "" {
			out("Phone:   %s\n", c.Phone)
		}
		if len(c.Tags) > 0 {
			out("Tags:    [%s]\n", strings.Join(c.Tags, ", "))
		}
		if len(c.Fields) > 0 {
			outln("Fields:")
			for k, v := range c.Fields {
				out("  %s: %v\n", k, v)
			}
		}
		out("Created: %s\n", c.CreatedAt.Format(time.RFC3339))
		out("Updated: %s\n", c.UpdatedAt.Format(time.RFC3339))

		// Show relationships
		rels, err := store.ListRelationships(c.ID)
		if err != nil {
			return err
		}
		if len(rels) > 0 {
			outln("Relationships:")
			for _, r := range rels {
				out("  %s -[%s]-> %s", cyan.Sprint(r.SourceID), r.Type, cyan.Sprint(r.TargetID))
				if r.Context != "" {
					out(" (%s)", r.Context)
				}
				outln()
			}
		}
		return nil
	},
}

var contactEditCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "Edit an existing contact",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := resolveContact(args[0])
		if err != nil {
			return err
		}

		if cmd.Flags().Changed("name") {
			c.Name, _ = cmd.Flags().GetString("name")
		}
		if cmd.Flags().Changed("email") {
			c.Email, _ = cmd.Flags().GetString("email")
		}
		if cmd.Flags().Changed("phone") {
			c.Phone, _ = cmd.Flags().GetString("phone")
		}
		if cmd.Flags().Changed("field") {
			fields, _ := cmd.Flags().GetStringArray("field")
			for _, f := range fields {
				k, v, ok := strings.Cut(f, "=")
				if !ok {
					return fmt.Errorf("invalid field format %q, expected KEY=VALUE", f)
				}
				c.Fields[k] = v
			}
		}
		if cmd.Flags().Changed("tag") {
			c.Tags, _ = cmd.Flags().GetStringSlice("tag")
		}

		c.Touch()
		if err := store.UpdateContact(c); err != nil {
			return err
		}

		out("Updated contact %s\n", color.New(color.FgCyan).Sprint(c.ID))
		return nil
	},
}

var contactRmCmd = &cobra.Command{
	Use:     "rm <id>",
	Aliases: []string{"delete", "del"},
	Short:   "Remove a contact",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := resolveContact(args[0])
		if err != nil {
			return err
		}

		if err := store.DeleteContact(c.ID); err != nil {
			return err
		}

		out("Deleted contact %s\n", color.New(color.FgCyan).Sprint(c.ID))
		return nil
	},
}

// resolveContact looks up a contact by full UUID or ID prefix.
func resolveContact(idStr string) (*models.Contact, error) {
	if id, err := uuid.Parse(idStr); err == nil {
		return store.GetContact(id)
	}
	return store.GetContactByPrefix(idStr)
}

func init() {
	contactAddCmd.Flags().String("email", "", "contact email address")
	contactAddCmd.Flags().String("phone", "", "contact phone number")
	contactAddCmd.Flags().StringArray("field", nil, "custom field as KEY=VALUE (repeatable)")
	contactAddCmd.Flags().StringSlice("tag", nil, "tag to apply (repeatable)")

	contactListCmd.Flags().StringP("tag", "t", "", "filter by tag")
	contactListCmd.Flags().StringP("search", "s", "", "search contacts")
	contactListCmd.Flags().IntP("limit", "n", 20, "max results to show")

	contactEditCmd.Flags().String("name", "", "new name")
	contactEditCmd.Flags().String("email", "", "new email")
	contactEditCmd.Flags().String("phone", "", "new phone")
	contactEditCmd.Flags().StringArray("field", nil, "set field KEY=VALUE (repeatable)")
	contactEditCmd.Flags().StringSlice("tag", nil, "replace tags (repeatable)")

	contactCmd.AddCommand(contactAddCmd)
	contactCmd.AddCommand(contactListCmd)
	contactCmd.AddCommand(contactShowCmd)
	contactCmd.AddCommand(contactEditCmd)
	contactCmd.AddCommand(contactRmCmd)
	rootCmd.AddCommand(contactCmd)
}
