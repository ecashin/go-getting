// Word count plus English translation!
//
// Usage is demonstrated by a command-line transcript.  The shell
// built-in, echo, writes "a b b c c c" plus a newline character to
// its standard output.  Because of the vertical bar, the shell uses
// the pipe system call to connect that standard output to the
// standard input of the second command.
// 
// The second command, "go run wc.go", tells go to compile the Go
// source file, wc.go, and then to run the compiled binary.
//
// The compiled binary prints the list of most-common words inside
// square brackets.
//
// In the first run, there's one mode, and in the second run, there
// are two.
// 
// ecashin@montgomery:~/git/go-getting$ echo a b b c c c | go run wc.go
// [c]
// ecashin@montgomery:~/git/go-getting$ echo d d d a b b c c c | go run wc.go
// [d c]
// ecashin@montgomery:~/git/go-getting$ 


// This Go source file implements resources inside the "main" package,
// so that it can be the core of a running program.
package main

// For this program, three of the core packages of the Go standard
// library are used.  Importing them brings them into the program's
// namespace.
import (
	"bufio"			// the buffered input/output package
	"fmt"			// the formatted string package
	"os"			// the operating system interface package
)

// The main function is where the Go runtime starts executing the program.
func main() {
	// The colon-equals syntax allows simultaneous variable
	// declaration and initialization with type inference.  The
	// type of scanner is whatever is returned by the bufio
	// package's NewScanner function.  The scanner is initialized
	// to use standard input from the operating system.
	scanner := bufio.NewScanner(os.Stdin)

	// The scanner is configured to split the input stream based
	// on words.
	scanner.Split(bufio.ScanWords)

	// The built-in routine, make, is used to define a map.  The
	// map acts as a dictionary with keys of type string and
	// values of type int.
	counts := make(map[string]int)

	// The only looping construct in Go is "for".  This loop is
	// like a Java "while" loop, because it only has one
	// expression after "for" and before the loop body.
	//
	// The .Scan method of the bufio scanner returns true while
	// there's more input to see.
	for scanner.Scan() {
		// The post-fix increment operator, plus-plus, is more
		// limited in Go than in C.  But it still adds one.
		//
		// In Go, heavy use is made of the convention that
		// data without explicit initialization is its zero
		// value.  For the integers in this map, that's a
		// zero.  The scanner.Text() call returns the current
		// word, and it is used as a key in the dictionary.
		// The corresponding value is then incremented.
		counts[scanner.Text()]++
	}

	// Declare an integer for storing the maximum count.
	max := 0

	// Declare a slice of string with zero elements.  (Just think
	// of it as an array for now.)  It will store the modes.
	modes := make([]string, 0)

	// This loop goes over every key/value pair in the map.
	//
	// The keys will be words, like "a", "b", and "c" in the
	// transcript at the top of this file.  The values will be the
	// counts, like 1, 2, and 3 in the same transcript.
	//
	// The .Println function in the fmt package is like Java's
	// System.out.Println.
	for word, count := range counts {
		// If the current counter is larger than the maximum
		// count seen so far, then reset the list of modes
		// with a slice containing only this word.  Record the
		// new maximum.
		if count > max {
			modes = []string{word}
			max = count
		} else if count == max {
			// Append any other word with the same maximal
			// count to the mode list.
			modes = append(modes, word)
		}
	}

	// Print a line to standard output using the default
	// stringification for the value.
	fmt.Println(modes)
}
