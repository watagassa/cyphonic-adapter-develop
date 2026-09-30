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
	"fmt"
	"runtime"
)

const (
	Version   = "0.3.2"                                    // Supports automatic recovery
	CodeOwner = "r0719en@pluslab.org, mitsuki@pluslab.org" // Code owner, or Maintainer
)

func showVersion() {
	fmt.Printf("CYPHONIC adapter daemon v%s\n\nUserspace CYPHONIC adapter daemon for %s-%s.\nInformation available at https://cyphonic.esa.io/posts/51\nCopyright (C) @Pluslab, Lab. <%s>.\n", Version, runtime.GOOS, runtime.GOARCH, CodeOwner)
}
