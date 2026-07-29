package root

import (
	"testing"

	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestKeepStdoutCleanForFlagValueRedirectsStdoutLogsOnMatch(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("format", "", "")
	assert.NoError(t, cmd.Flags().Set("format", "docker"))
	KeepStdoutCleanForFlagValue(cmd, "format", "docker")

	logFlags := &logFlags{file: flags.NewLogFileFlag()}
	assert.NoError(t, logFlags.file.Set("stdout"))
	assert.NoError(t, logFlags.keepStdoutClean(cmd))
	assert.Equal(t, "stderr", logFlags.file.String())
}

func TestKeepStdoutCleanForFlagValueLeavesStdoutLogsOnMismatch(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("format", "", "")
	assert.NoError(t, cmd.Flags().Set("format", "json"))
	KeepStdoutCleanForFlagValue(cmd, "format", "docker")

	logFlags := &logFlags{file: flags.NewLogFileFlag()}
	assert.NoError(t, logFlags.file.Set("stdout"))
	assert.NoError(t, logFlags.keepStdoutClean(cmd))
	assert.Equal(t, "stdout", logFlags.file.String())
}
