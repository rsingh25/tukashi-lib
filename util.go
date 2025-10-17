// utils.go
package mymodule

import "fmt"

// PrintGreeting prints a standardized greeting message
func PrintGreeting(name string) {
	fmt.Printf("Hello, %s! Welcome to the custom module.\n", name)
}

// Double returns the input integer multiplied by two
func Double(x int) int {
	return x * 2
}
