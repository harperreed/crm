// ABOUTME: CLI commands for managing CRM companies.
// ABOUTME: Provides add, list, show, edit, and remove subcommands under "company".

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

var companyCmd = &cobra.Command{
	Use:     "company",
	Aliases: []string{"co"},
	Short:   "Manage companies",
}

var companyAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new company",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c := models.NewCompany(args[0])

		domain, _ := cmd.Flags().GetString("domain")
		fields, _ := cmd.Flags().GetStringSlice("field")
		tags, _ := cmd.Flags().GetStringSlice("tag")

		c.Domain = domain

		for _, f := range fields {
			k, v, ok := strings.Cut(f, "=")
			if !ok {
				return fmt.Errorf("invalid field format %q, expected KEY=VALUE", f)
			}
			c.Fields[k] = v
		}
		c.Tags = tags

		if err := store.CreateCompany(c); err != nil {
			return err
		}

		cyan := color.New(color.FgCyan)
		out("Created company %s\n", cyan.Sprint(c.ID))
		return nil
	},
}

var companyListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls", "l"},
	Short:   "List companies",
	RunE: func(cmd *cobra.Command, args []string) error {
		tag, _ := cmd.Flags().GetString("tag")
		search, _ := cmd.Flags().GetString("search")
		limit, _ := cmd.Flags().GetInt("limit")

		filter := &storage.CompanyFilter{
			Search: search,
			Limit:  limit,
		}
		if tag != "" {
			filter.Tag = &tag
		}

		companies, err := store.ListCompanies(filter)
		if err != nil {
			return err
		}

		if len(companies) == 0 {
			outln("No companies found.")
			return nil
		}

		cyan := color.New(color.FgCyan)
		bold := color.New(color.Bold)
		for _, c := range companies {
			out("%s  %s", cyan.Sprint(c.ID), bold.Sprint(c.Name))
			if c.Domain != "" {
				out("  <%s>", c.Domain)
			}
			if len(c.Tags) > 0 {
				out("  [%s]", strings.Join(c.Tags, ", "))
			}
			outln()
		}
		return nil
	},
}

var companyShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show company details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := resolveCompany(args[0])
		if err != nil {
			return err
		}

		cyan := color.New(color.FgCyan)
		bold := color.New(color.Bold)

		out("ID:      %s\n", cyan.Sprint(c.ID))
		out("Name:    %s\n", bold.Sprint(c.Name))
		if c.Domain != "" {
			out("Domain:  <%s>\n", c.Domain)
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

var companyEditCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "Edit an existing company",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := resolveCompany(args[0])
		if err != nil {
			return err
		}

		if cmd.Flags().Changed("name") {
			c.Name, _ = cmd.Flags().GetString("name")
		}
		if cmd.Flags().Changed("domain") {
			c.Domain, _ = cmd.Flags().GetString("domain")
		}
		if cmd.Flags().Changed("field") {
			fields, _ := cmd.Flags().GetStringSlice("field")
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
		if err := store.UpdateCompany(c); err != nil {
			return err
		}

		out("Updated company %s\n", color.New(color.FgCyan).Sprint(c.ID))
		return nil
	},
}

var companyRmCmd = &cobra.Command{
	Use:     "rm <id>",
	Aliases: []string{"delete", "del"},
	Short:   "Remove a company",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := resolveCompany(args[0])
		if err != nil {
			return err
		}

		if err := store.DeleteCompany(c.ID); err != nil {
			return err
		}

		out("Deleted company %s\n", color.New(color.FgCyan).Sprint(c.ID))
		return nil
	},
}

// resolveCompany looks up a company by full UUID or ID prefix.
func resolveCompany(idStr string) (*models.Company, error) {
	if id, err := uuid.Parse(idStr); err == nil {
		return store.GetCompany(id)
	}
	return store.GetCompanyByPrefix(idStr)
}

func init() {
	companyAddCmd.Flags().String("domain", "", "company domain")
	companyAddCmd.Flags().StringSlice("field", nil, "custom field as KEY=VALUE (repeatable)")
	companyAddCmd.Flags().StringSlice("tag", nil, "tag to apply (repeatable)")

	companyListCmd.Flags().StringP("tag", "t", "", "filter by tag")
	companyListCmd.Flags().StringP("search", "s", "", "search companies")
	companyListCmd.Flags().IntP("limit", "n", 20, "max results to show")

	companyEditCmd.Flags().String("name", "", "new name")
	companyEditCmd.Flags().String("domain", "", "new domain")
	companyEditCmd.Flags().StringSlice("field", nil, "set field KEY=VALUE (repeatable)")
	companyEditCmd.Flags().StringSlice("tag", nil, "replace tags (repeatable)")

	companyCmd.AddCommand(companyAddCmd)
	companyCmd.AddCommand(companyListCmd)
	companyCmd.AddCommand(companyShowCmd)
	companyCmd.AddCommand(companyEditCmd)
	companyCmd.AddCommand(companyRmCmd)
	rootCmd.AddCommand(companyCmd)
}
