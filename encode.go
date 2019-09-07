package supd

import (
	"fmt"
	"io/ioutil"

	"github.com/gwaycc/supd/config"

	"github.com/gwaylib/errors"
)

type EncodeCommand struct {
	Key string `short:"k" long:"key" description:"Auth key input"`
	Out string `short:"o" long:"out" description:"File ouput"`
}

func (v EncodeCommand) Execute(args []string) error {
	if len(args) < 1 {
		fmt.Println(`ErrCommand: supd encode [option] input-file`)
		return nil
	}
	if v.Key != string(config.ConfKey) {
		fmt.Println("Key not match")
		return nil
	}
	fileData, err := ioutil.ReadFile(args[0])
	if err != nil {
		fmt.Println(errors.As(err))
		return nil
	}
	output := config.Encode(fileData, config.ConfKey)
	if len(v.Out) > 0 {
		if err := ioutil.WriteFile(v.Out, output, 0666); err != nil {
			fmt.Println(errors.As(err))
			return nil
		}
		fmt.Println("has output to: " + v.Out)
		return nil
	}
	fmt.Println(string(output))

	return nil
}

type DecodeCommand struct {
	Key string `short:"k" long:"key" description:"Auth key input"`
	Out string `short:"o" long:"out" description:"File ouput"`
}

func (v DecodeCommand) Execute(args []string) error {
	if len(args) < 1 {
		fmt.Println(`ErrCommand: supd decode [option] input-file`)
		return nil
	}
	if v.Key != string(config.ConfKey) {
		fmt.Println("Key not match")
		return nil
	}
	fileData, err := ioutil.ReadFile(args[0])
	if err != nil {
		fmt.Println(errors.As(err))
		return nil
	}
	output, err := config.Decode(fileData, config.ConfKey)
	if err != nil {
		fmt.Println(errors.As(err))
		return nil
	}
	if len(v.Out) > 0 {
		if err := ioutil.WriteFile(v.Out, output, 0666); err != nil {
			fmt.Println(errors.As(err))
			return nil
		}
		fmt.Println("has output to: " + v.Out)
		return nil
	}
	fmt.Println(string(output))
	return nil
}

func init() {
	parser.AddCommand(
		"encode",
		"encode ini to dat",
		"encode ini to dat",
		&EncodeCommand{},
	)
	parser.AddCommand(
		"decode",
		"decode dat to ini",
		"decode dat to ini",
		&DecodeCommand{},
	)

}
