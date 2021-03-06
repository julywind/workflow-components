package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"

	gomail "gopkg.in/gomail.v2"
)

const STAGE_TYPE_END = "end"

type Builder struct {
	FromUser     string
	Secret       string
	ToUsers      string
	Subject      string
	Type         string
	Server       string
	Port         string
	Body         string
	EmailContent EmailContent
}

func NewBuilder(envs map[string]string) (*Builder, error) {
	b := &Builder{}
	if envs["FROM_USER"] == "" {
		return nil, fmt.Errorf("environment variable FROM_USER is requried")
	} else {
		b.FromUser = envs["FROM_USER"]
		//fmt.Println(b.FromUser)
	}
	if envs["TO_USERS"] == "" {
		return nil, fmt.Errorf("environment variable TO_USER is requried")
	} else {
		b.ToUsers = envs["TO_USERS"]
		//fmt.Println(b.ToUsers)
	}
	if envs["SECRET"] == "" {
		return nil, fmt.Errorf("environment variable SECRET is requried")
	} else {
		b.Secret = envs["SECRET"]
		//fmt.Println(b.Secret)
	}
	if envs["SMTP_SERVER_PORT"] == "" {
		return nil, fmt.Errorf("environment variable SMTP_SERVER_PORT is requried")
	} else {
		//fmt.Println(envs["SMTP_SERVER_PORT"])
		param := strings.SplitN(envs["SMTP_SERVER_PORT"], ":", 2)
		b.Server = param[0]
		b.Port = param[1]
		fmt.Printf("smtp_server: %s, smtp_port: %s\n", b.Server, b.Port)
	}

	b.Subject = envs["SUBJECT"]

	if envs["TEXT"] != "" {
		b.Body = envs["TEXT"]
		return b, nil
	}

	task := &FlowTask{}
	err := json.Unmarshal([]byte(envs["_WORKFLOW_TASK_DETAIL"]), task)
	if err != nil {
		return nil, err
	}

	fmt.Printf("show: %+v\n", task)

	data, err := ParseTemplate(task)
	if err != nil {
		return nil, err
	}
	b.Body = data

	return b, nil
}

func (b *Builder) run() error {
	if err := b.SendEmail(); err != nil {
		log.Printf("Failed to send the email to %s\n", b.ToUsers)
		return err
	} else {
		log.Printf("Email has been sent to %s\n", b.ToUsers)
	}
	return nil
}

func ParseTemplate(data *FlowTask) (string, error) {
	var fileName = "/usr/bin/template.html"

	t, err := template.New("template.html").Funcs(template.FuncMap{"myFunc": myFunc, "totalTime": timeConsuming}).ParseFiles(fileName)
	if err != nil {
		return "", err
	}
	buffer := new(bytes.Buffer)
	if err := t.Execute(buffer, data); err != nil {
		return "", err
	}
	return buffer.String(), nil
}

func myFunc(stage Stage) template.HTML {
	var mdText = ""
	nm := stage.Name
	status := stage.Status
	stageType := stage.Type
	jobs := stage.Jobs
	if stageType != STAGE_TYPE_END {
		mdText += fmt.Sprintf(" %s : %s <br>", nm, status)

		for _, job := range jobs {
			name := job.Name
			status := job.Status
			mdText += fmt.Sprintf("&nbsp &nbsp &nbsp &nbsp %s : %s <br>", name, status)
		}
	}

	return template.HTML(mdText)
}
func timeConsuming(start *time.Time, end *time.Time) string {
	var totalTime string
	if start != nil && end != nil {
		totalTime = fmt.Sprintf("总耗时: %d 秒", (int64)(end.Sub(*start).Seconds()))
	}
	return totalTime
}

func (b *Builder) SendEmail() error {
	var toUsers = strings.Split(b.ToUsers, ",")
	m := gomail.NewMessage()
	//设置发件人
	m.SetAddressHeader("From", b.FromUser, "工作流邮件通知")
	//设置收件人
	m.SetHeader("To", toUsers...)
	//设置主题
	m.SetHeader("Subject", b.Subject)
	//设置正文
	//m.SetBody("text", "hello world!")

	m.SetBody("text/html", b.Body)
	//设置发送邮件服务器、端口、发件人账号、发件人密码
	port, err := strconv.Atoi(b.Port)
	if err != nil {
		return err
	}
	d := gomail.NewPlainDialer(b.Server, port, b.FromUser, b.Secret)

	if err := d.DialAndSend(m); err != nil {
		return err
	}
	return nil
}

type CMD struct {
	Command []string
	WorkDir string
}

func (c CMD) Run() (string, error) {
	fmt.Println("Run CMD: ", strings.Join(c.Command, "工作流通知"))

	cmd := exec.Command(c.Command[0], c.Command[1:]...)
	if c.WorkDir != "" {
		cmd.Dir = c.WorkDir
	}
	data, err := cmd.CombinedOutput()

	result := string(data)
	if len(result) > 0 {
		fmt.Println(result)

	}

	return result, err
}
