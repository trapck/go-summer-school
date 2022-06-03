package main

import (
	"os"

	"wasfaty.api/services/mpi/cmd"
)

func main() {
	if err := cmd.Run(); err != nil {
		os.Exit(18)
	}
}
