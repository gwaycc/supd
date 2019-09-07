package supd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/gwaycc/supd/config"

	"github.com/gwaylib/errors"
)

type ChmodCommand struct {
	Key string `short:"k" long:"key" description:"Auth key input"`
}

func (v ChmodCommand) Execute(args []string) error {
	if len(args) < 2 {
		fmt.Println(`ErrCommand: supd chmod value file`)
		return nil
	}
	if v.Key != string(config.ConfKey) {
		fmt.Println("Key not match")
		return nil
	}
	mod, err := strconv.ParseUint(args[0], 8, 32)
	if err != nil {
		fmt.Println(errors.As(err, args[0]))
		return nil
	}
	fileMode := os.FileMode(mod)
	if err := os.Chmod(args[1], fileMode); err != nil {
		fmt.Println(errors.As(err, args))
		return nil
	}
	fmt.Printf("chmod %s done\n", fileMode)
	return nil
}

func init() {
	parser.AddCommand(
		"chmod",
		"chnage file mode",
		"chnage file mode",
		&ChmodCommand{},
	)
}
