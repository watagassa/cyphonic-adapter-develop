package domain

import (
	"fmt"
	"os"
)

func StartingMessage() {
	fmt.Println()
	fmt.Fprintln(os.Stdout, "┌───────────────────────────────────────────────────────┐")
	fmt.Fprintln(os.Stdout, "│                                                       │")
	fmt.Fprintln(os.Stdout, "│   Starting the CYPHONIC Adapterd...                   │")
	fmt.Fprintln(os.Stdout, "│   Please check the running signal and                 │")
	fmt.Fprintln(os.Stdout, "│   then connect the general node to the network hub.   │")
	fmt.Fprintln(os.Stdout, "│                                                       │")
	fmt.Fprintln(os.Stdout, "│   Click here for details:                             │")
	fmt.Fprintln(os.Stdout, "│      https://cyphonic.esa.io/posts/51                 │")
	fmt.Fprintln(os.Stdout, "│                                                       │")
	fmt.Fprintln(os.Stdout, "└───────────────────────────────────────────────────────┘")
	fmt.Println()
}

func ProvisionedMessage() {
	fmt.Println()
	fmt.Fprintln(os.Stdout, "┌───────────────────────────────────────────────────────┐")
	fmt.Fprintln(os.Stdout, "│          ▓▒░ Adapter Device provisioned ▓▒░           │")
	fmt.Fprintln(os.Stdout, "└───────────────────────────────────────────────────────┘")
	fmt.Println()
}

func Version4Message() {
	fmt.Println()
	fmt.Fprintln(os.Stdout, "┌───────────────────────────────────────────────────────┐")
	fmt.Fprintln(os.Stdout, "│                       IPv4 Mode                       │")
	fmt.Fprintln(os.Stdout, "└───────────────────────────────────────────────────────┘")
	fmt.Println()
}

func Version6Message() {
	fmt.Println()
	fmt.Fprintln(os.Stdout, "┌───────────────────────────────────────────────────────┐")
	fmt.Fprintln(os.Stdout, "│                       IPv6 Mode                       │")
	fmt.Fprintln(os.Stdout, "└───────────────────────────────────────────────────────┘")
	fmt.Println()
}
