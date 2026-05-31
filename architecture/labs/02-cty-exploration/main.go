package main

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
)

// This lab demonstrates HashiCorp's go-cty, a dynamic type system for Go.
// We explore how it handles polymorphic variables, strict type comparisons,
// invalid operations, and "Unknown" values (vital for Dry-Runs).

func main() {
	fmt.Println("=== LAB 02: HashiCorp go-cty Exploration ===")

	// 1. Creating Typed Dynamic Values
	// cty has its own set of Types and Values that wrap Go primitives.
	envVal := cty.StringVal("production")
	replicasVal := cty.NumberIntVal(5)
	maxReplicasVal := cty.NumberIntVal(10)

	fmt.Printf("\n--- 1. Created Dynamic Values ---\n")
	fmt.Printf("env: type=%s, value=%#v\n", envVal.Type().FriendlyName(), envVal.AsString())
	fmt.Printf("replicas: type=%s, value=%v\n", replicasVal.Type().FriendlyName(), replicasVal.AsBigFloat())

	// 2. Safe and Typed Operations
	// Let's compare replicas (5) < maxReplicas (10)
	// cty operations return a cty.Value representing the result (a boolean).
	fmt.Printf("\n--- 2. Comparisons (replicas < maxReplicas) ---\n")
	resultVal := replicasVal.LessThan(maxReplicasVal)
	fmt.Printf("Result value: type=%s, value=%v\n", resultVal.Type().FriendlyName(), resultVal.True())

	// 3. Error Interception on Invalid Operations
	// What happens if we try to check if replicas (Number) > env (String)?
	// In standard Go interface{}, comparing "production" > 5 causes a runtime panic.
	// In cty, it is caught safely at the type level.
	fmt.Printf("\n--- 3. Type Mismatch Handling ---\n")
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("Caught expected runtime type safety panic: %v\n", r)
			}
		}()

		// This will panic or return a type error because you cannot compare string and number.
		// cty.Value methods panic on invalid operations to enforce strict correctness,
		// allowing us to intercept type mismatches cleanly before running mutations.
		_ = replicasVal.GreaterThan(envVal)
	}()

	// 4. The Superpower: "Unknown" Values (Dry-Run / Static Checks)
	// In Terraform/ast-mutate, we might want to validate a playbook without knowing
	// the actual input variable value yet (or if a value is dynamically computed by another step).
	// We can represent this as a cty.UnknownVal of a specific type.
	fmt.Printf("\n--- 4. Unknown Values (Dry-run mode) ---\n")

	unknownEnv := cty.UnknownVal(cty.String) // We know it's a String, but not its value yet.
	targetProd := cty.StringVal("production")

	// Let's check: unknownEnv == "production"
	isProdResult := unknownEnv.Equals(targetProd)

	fmt.Printf("Is unknownEnv == 'production'?\n")
	fmt.Printf("Result type: %s\n", isProdResult.Type().FriendlyName())
	fmt.Printf("Is result Raw/Known? %v\n", isProdResult.IsKnown())
	// Because the input env was Unknown, the equality result is also Unknown!
	// This allows the evaluation engine to skip executing branches that depend on unknown values
	// during a Dry-Run, without failing with a nil-pointer exception or type mismatch.
}
