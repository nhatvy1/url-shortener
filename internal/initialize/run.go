package initialize

import "fmt"

func Run() {
	fmt.Println("Start")
	defer fmt.Println("Deferred")

	fmt.Println("End")
}
