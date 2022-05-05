package transfer

import (
	"github.com/spf13/cobra"
)

func AddSpaceVar(cmd *cobra.Command) {
	// space := string(types.SPACE_NATIVE)
	cmd.PersistentFlags().StringVar(&space, "space", "", "Space name")
}
