package k8s

import (
    "fmt"
    "os/exec"
    "os"
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


func Cross_App_Check(ns string, rs_n string) string{
	log.Println(ns, rs_n)
	cmd := exec.Command("kubectl", "-n", ns ,"get", "deploy", rs_n)
        stdout1, err := cmd.Output()
        log.Println(cmd, err)
        if err != nil{
                return "TRUE"
        }
        log.Println(stdout1)
        return("FALSE")
}

func Cross_Check(ns string) string{
	cmd := exec.Command("kubectl", "get", "ns", ns)
	stdout1, err := cmd.Output()
	log.Println(err)
	if err != nil{
		return "TRUE"
	}
	log.Println(stdout1)
	return("FALSE")
}
func Cross_Create(ns string) string{
	log.Println(ns)
	cmd := exec.Command("kubectl", "create", "ns", ns)
	stdout1,err := cmd.Output()
	if err != nil{
		return err.Error()
	}
	return(string(stdout1))
}

//type Message map[string]interface{}
func Cross_Get(ns string, rs_n string) string{
	log.Println(ns, rs_n)
	//if str, ok := msg["namespace"].(string); ok{
	cmd := exec.Command("kubectl", "-n", ns, "get", "deploy", rs_n)
//}
	stdout1, err := cmd.Output()
	if err !=nil {
		log.Println(err.Error())
		return err.Error()
	}
	return(string(stdout1))

}


func Cross_Apply(ns string, rs_n string, rs string)string{

	 f, err := os.Create("dep.json")

	    if err != nil {
        	log.Fatal(err)
   	 }

    	 defer f.Close()

    	 _, err2 := f.WriteString(rs)

  	if err2 != nil {
       	 log.Fatal(err2)
   	 }
	 cmd := exec.Command("kubectl", "-n", ns, "apply", "-f", "dep.json")
	 stdout1, err3 := cmd.Output()
	 if err3 != nil{
		 return err3.Error()
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
