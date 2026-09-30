// MIT License

// Copyright (c) 2023 Pluslab at AIT

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package main

import (
	"log"
	"time"

	"github.com/Pluslab/cyphonic-adapter/adapterd/controller"
	"github.com/spf13/cobra"
)

func main() {
	var cmd = &cobra.Command{Use: "cadctl"}

	// $ /bin/sh cactl start
	var cmdStart = &cobra.Command{
		Use:   "start",
		Short: "Start the CYPHONIC adapter daemon service",
		Run: func(cmd *cobra.Command, args []string) {
			tz := time.FixedZone("JST", 0)
			time.Local = tz
			status, err := controller.Run()
			if err != nil {
				log.Fatalf("\n[INFO] process code: %d \n status reason: %v\n", status, err)
			}
		},
	}

	// $ /bin/sh cactl version
	var cmdVersion = &cobra.Command{
		Use:   "version",
		Short: "Show the CYPHONIC adapter daemon version information",
		Run: func(cmd *cobra.Command, args []string) {
			showVersion()
		},
	}

	cmd.AddCommand(cmdStart)
	cmd.AddCommand(cmdVersion)

	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}
