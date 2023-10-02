package openstack

import (
	"fmt"
	"os/exec"
)

func Get() string {
	app := "openstack"

	arg0 := "image"
	arg1 := "show"
	arg2 := "testimagehi"
	//arg3 := "golang"

	cmd := exec.Command(app, arg0, arg1, arg2) //, arg2, arg3)
	stdout1, err := cmd.Output()

	if err != nil {
		fmt.Println(err.Error())
		return ""
	}

	// Print the output
	return (string(stdout1))
}

func Deploy() string {
	app := "openstack"

	arg0 := "image"
	arg1 := "create"
	arg2 := "testimagehi"
	//arg3 := "golang"

	cmd := exec.Command(app, arg0, arg1, arg2) //, arg3)
	stdout1, err := cmd.Output()

	if err != nil {
		return err.Error()
	}

	// Print the output
	return (string(stdout1))

}
