/*
Package cli provides command-line interface utilities for Mercator Jupiter.

The cli package includes output formatters, progress reporters, and common CLI
helpers used by the mercator command.

Output Formatting:

The cli package supports multiple output formats (text, JSON, CSV) for
displaying command results:

	formatter := cli.NewFormatter(cli.FormatJSON)
	data := MyCommandResult{...}
	if err := formatter.FormatTo(os.Stdout, data); err != nil {
		return err
	}

Progress Reporting:

For long-running operations, use the progress reporter:

	progress := cli.NewProgressReporter(os.Stdout)
	progress.Start(totalItems)
	for i := 0; i < totalItems; i++ {
		// Do work
		progress.Update(i + 1)
	}
	progress.Finish()

Signal Handling:

For graceful shutdown on SIGINT/SIGTERM:

	ctx := cli.SetupSignalHandler()
	// Use ctx for operations that should be cancelled on shutdown
*/
package cli
