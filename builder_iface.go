package main

// BuildArchiverGraph is set by the ticket that ships the graph generator
// (T-3+). Until it is set, the acceptance test t.Skips. The argument is
// the absolute path to a yatool_orig source tree. The return is the
// JSON-encoded graph in the same shape as sg.json.
var BuildArchiverGraph func(yatoolOrig string) ([]byte, error)
