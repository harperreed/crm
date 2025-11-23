# Scenario-Driven Development Guidelines (For Codegen Agents)

When generating or modifying code in this repository, follow these rules:

### 1. Do **not** produce unit tests

Avoid generating unit tests entirely. Your typical mistakes are conceptual misunderstandings, not small implementation errors. Unit tests add noise and slow iteration.

### 2. Validate features only through **end-to-end scenarios**

When you need to show that something works, generate a full flow that exercises the entire system from the outside—using public APIs, CLIs, handlers, or other stable interfaces. Treat the whole product as the meaningful unit of correctness.

### 3. Put all scenario programs in **`.scratch/`**

When demonstrating or checking behavior, write a “proof program” into the `.scratch/` directory.

* `.scratch/` is ignored by git.
* These proof programs are temporary.
* Generate a new one each cycle; never rely on old ones.

### 4. Promote recurring scenarios into **`scenarios.jsonl`**

If a scenario pattern shows up repeatedly across iterations, extract it into a structured entry in `scenarios.jsonl`. This file represents the canonical behaviors the system must support.

### 5. Expect verification by an **external scenario runner**

Assume the product is exercised from outside the codebase using stable hooks.
Write code so it can be driven externally, and produce clear, human-readable error messages when expectations aren’t met.
