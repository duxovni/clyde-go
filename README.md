* clyde

Clyde is a Markov chain chatbot created by cat@mit.edu.  He's been a
beloved presence on MIT Zephyr for ~20 years, amusing and delighting
many generations of students.  Clyde's perl code has been getting a
bit long in the tooth, so I wrote this port in order to add some
features that were hard to do in perl, make future maintenance and
development easier, and get my feet wet in Go.

`clyde-go` is currently under heavy development, and isn't really a
functional chatbot quite yet.

** Instructions

*** Installation

    $ go get github.com/sdukhovni/clyde-go/cmd/clyde

*** Usage

    $ $GOPATH/bin/clyde
