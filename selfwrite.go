package main
import "fmt"

var me = `package main
import "fmt"

var me = %c%s%c
func main() {
	fmt.Print(fmt.Sprintf(me, '\x60', me, '\x60'))
}
`
func main() {
	fmt.Print(fmt.Sprintf(me, '\x60', me, '\x60'))
}
