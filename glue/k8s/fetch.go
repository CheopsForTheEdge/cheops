package k8s

import (
    "fmt"
    "os/exec"
)

func Get() string {
    app := "kubectl"

    arg0 := "get"
    arg1 := "po"
    //arg2 := "\n\tfrom"
    //arg3 := "golang"

    cmd := exec.Command(app, arg0, arg1)//, arg2, arg3)
    stdout1, err := cmd.Output()

    if err != nil {
        fmt.Println(err.Error())
	return "HI"
    }

    // Print the output
    return(string(stdout1))
}

func Deploy() string {
	app := "kubectl"
	arg0 := "apply"
    arg1 := "-f"
    arg2 := "deployment.json"
    //arg3 := "golang"

    cmd := exec.Command(app, arg0, arg1, arg2)//, arg3)
	stdout1, err := cmd.Output()

	if err != nil {
        	return err.Error()
    	}

    // Print the output
    	return(string(stdout1))


}
