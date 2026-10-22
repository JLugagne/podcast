// Command podcast runs the podcast catalog API and its admin UI from a single
// binary.
package main

import "github.com/JLugagne/podcast/cmd/podcast"

func main() {
	podcast.Execute()
}
