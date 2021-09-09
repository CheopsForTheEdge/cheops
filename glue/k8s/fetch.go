package k8s

import (
    "fmt"
    "os/exec"
    "log"
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


func Get_Deploy(dep_name string) string{
	log.Println(dep_name)
	cmd := exec.Command("kubectl", "get", "deploy", dep_name, "-o", "json")
	stdout1, err := cmd.Output()

	if err != nil {
		log.Println(err.Error())
		return "err"
	}
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
