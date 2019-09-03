package supd

import (
	"fmt"
	"io/ioutil"

	"github.com/gwaycc/supd/config"

	"github.com/gwaylib/errors"
)

type EncodeCommand struct {
}

func (v EncodeCommand) Execute(args []string) error {
	if len(args) < 2 {
		fmt.Println(`ErrCommand: supd encode "key" input-file [output-file]`)
		return nil
	}
	if args[0] != string(config.ConfKey) {
		fmt.Println("Key not match")
		return nil
	}
	fileData, err := ioutil.ReadFile(args[1])
	if err != nil {
		fmt.Println(errors.As(err))
		return nil
	}
	output := config.Encode(fileData, config.ConfKey)
	if len(args) > 2 {
		outputFile := args[2]
		if err := ioutil.WriteFile(outputFile, output, 0666); err != nil {
			fmt.Println(errors.As(err))
			return nil
		}
		fmt.Println("has output to: " + outputFile)
		return nil
	}
	fmt.Println(string(output))

	return nil
}

type DecodeCommand struct {
}

func (v DecodeCommand) Execute(args []string) error {
	fmt.Println(args)
	if len(args) < 2 {
		fmt.Println(`error command. example:supd decode "key" inputfile [outputfile]`)
		return nil
	}
	if args[0] != string(config.ConfKey) {
		fmt.Println("Key not match")
		return nil
	}
	fileData, err := ioutil.ReadFile(args[1])
	if err != nil {
		fmt.Println(errors.As(err))
		return nil
	}
	output, err := config.Decode(fileData, config.ConfKey)
	if err != nil {
		fmt.Println(errors.As(err))
		return nil
	}
	if len(args) > 2 {
		outputFile := args[2]
		if err := ioutil.WriteFile(outputFile, output, 0666); err != nil {
			fmt.Println(errors.As(err))
			return nil
		}
		fmt.Println("has output to: " + outputFile)
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
