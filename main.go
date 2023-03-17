package main

import (
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"

	"text/template"

	"github.com/mkideal/cli"
)

type Mail struct {
	Server   string   `json:"smtp"`
	User     string   `json:"user"`
	Password string   `json:"password"`
	Sender   string   `json:"sender"`
	Port     int      `json:"port"`
	To       []string `json:"to"`
}

type Config struct {
	Servers   []string `json:"servers"`
	Duration  int      `json:"duration"`
	Iteration int      `json:"iteration"`
	MailInfo  Mail     `json:"mail"`
}

type argT struct {
	Help bool `cli:"h,help" usage:"show help"`
	/*Servers   []string      `cli:"server" usage:"List of server"`
	Duration  clix.Duration `cli:"pause" usage:"Pause duration" dft:"10s"`
	Iteration int           `cli:"i,iteration" usage:"Number of iteration call, -1 means always" dft:"-1"`*/
	Config Config `cli:"f,file" usage:"Config file" parser:"jsonfile"`
}

func (argv *argT) AutoHelp() bool {
	return argv.Help
}

func main() {
	os.Exit(cli.Run(new(argT), func(ctx *cli.Context) error {
		argv := ctx.Argv().(*argT)
		if len(argv.Config.Servers) > 0 {
			callOnServers(argv.Config.Servers, time.Duration(argv.Config.Duration*int(time.Second)), argv.Config.Iteration, argv.Config.MailInfo)
			//} else if len(argv.Servers) > 0 {
			// callOnServers(argv.Servers, argv.Duration.Duration, argv.Iteration, _)
		} else {
			fmt.Printf("Servers list is mandatory")
		}

		return nil
	}))
}

func callOnServers(servers []string, pause time.Duration, iteration int, mailInfo Mail) {
	for i := 0; i < iteration || iteration == -1; i++ {
		errorServer := []string{}
		for _, server := range servers {

			req, err := http.NewRequest(http.MethodHead, server, nil)
			if err != nil {
				fmt.Printf("%s: could not create request: %s\n", server, err)
				continue
			}

			res, err := http.DefaultClient.Do(req)
			if err != nil {
				fmt.Printf("%s: error making http request: %s\n", server, err)
				errorServer = append(errorServer, server)

				continue
			}

			if res.StatusCode >= 200 && res.StatusCode < 300 {
				fmt.Printf("%s is OK\n", server)
			} else {
				fmt.Printf("client: status code: %d\n", res.StatusCode)
			}

		}
		if len(errorServer) > 0 {
			sendErrorMail(errorServer, mailInfo)
		}

		time.Sleep(pause)
	}
}

type ErrorServerInfo struct {
	From       string
	To         []string
	ServerList []string
}

func sendErrorMail(serverWithError []string, mailInfo Mail) {
	var serverError ErrorServerInfo
	serverError.ServerList = serverWithError
	serverError.To = mailInfo.To
	serverError.From = mailInfo.Sender

	auth := smtp.PlainAuth("", mailInfo.User, mailInfo.Password, mailInfo.Server)
	to := serverError.To
	ut, err := template.New("Mail error").Parse("From: {{ .From }}\r\n" +
		"To: {{ .To }}\r\n" +
		"Subject: Server not answering\r\n" +
		"\r\n" +
		"{{ range .ServerList }}" +
		" {{.}}\r\n" +
		"{{ end }}")
	if err != nil {
		panic(err)
	}
	msg := new(strings.Builder)
	errTemplate := ut.Execute(msg, serverError)
	if errTemplate != nil {
		panic(errTemplate)
	}
	fmt.Printf("%s %s", serverError.ServerList, msg.String())

	error := smtp.SendMail(mailInfo.Server+":"+strconv.Itoa(mailInfo.Port), auth, mailInfo.Sender, to, []byte(msg.String()))
	if error != nil {
		log.Fatal(error)
	}
}
