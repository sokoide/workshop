package main

import (
	"fmt"

	"golang.org/x/crypto/ssh"
)

func main() {
	// Deliberately uses a vulnerable version of x/crypto/ssh
	// Run 'govulncheck ./...' to detect vulnerabilities in this code
	config := &ssh.ClientConfig{
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	fmt.Printf("SSH config: %+v\n", config)
	fmt.Println("Run 'govulncheck ./...' to detect vulnerabilities in this code.")
}
