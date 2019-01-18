package cli

import (
	"bufio"
	"fmt"
	//"io"
	"io/ioutil"
	"net"
	"os"
	//"os/exec"
	//"os/signal"
	"strconv"
	//"strings"
	//"syscall"

	"golang.org/x/crypto/ssh"
	//"golang.org/x/crypto/ssh/terminal"

	//"github.com/kr/pty"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"

	"github.com/SamsungSLAV/boruta"
	util "github.com/SamsungSLAV/boruta/cli/ssh"
	"github.com/SamsungSLAV/slav/logger"
)

type ReqsAcquireCmd struct {
	Command  *cobra.Command
	Requests boruta.Requests
}

func NewReqsAcquireCmd(r boruta.Requests) *ReqsAcquireCmd {
	rc := &ReqsAcquireCmd{}
	rc.Requests = r
	rc.Command = &cobra.Command{
		Use:   "acquire",
		Short: "When request is IN PROGRESS...", //TODO
		Long:  "",                               //TODO
		Args:  cobra.ExactArgs(1),
		Run:   rc.AcquireReqs,
	}
	return rc
}

func (rc *ReqsAcquireCmd) AcquireReqs(cmd *cobra.Command, args []string) {
	home, err := homedir.Dir()
	identityFilePath := home + "/.ssh/dryad-identity"

	rid, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		logger.WithError(err).Errorf("Failed to parse %s to uint64", args[0])
	}
	accessInfo, err := rc.Requests.AcquireWorker(boruta.ReqID(rid))
	if err != nil {
		logger.WithError(err).Error("Failed to create request.")
	}
	//fmt.Println(accessInfo)
	//TODO: handle identity file in some more sensible way

	if err := util.CreateKeyFiles(&accessInfo.Key, identityFilePath); err != nil {
		logger.WithError(err).Error("Failed to create ssh key files")
	}

	_, port, err := net.SplitHostPort(accessInfo.Addr.String())

	buffer, err := ioutil.ReadFile(identityFilePath) //handle err
	key, err := ssh.ParsePrivateKey(buffer)

	sshConfig := &ssh.ClientConfig{
		User:            accessInfo.Username,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(key)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	connection, err := ssh.Dial("tcp", "106.120.47.240:"+port, sshConfig)
	if err != nil {
		logger.WithError(err).Error("Failed to dial ssh")
		return
	}

	session, err := connection.NewSession()
	if err != nil {
		logger.WithError(err).Error("Failed to dial ssh")
		return
	}

	defer func() {
		if err := session.Close(); err != nil {
			logger.WithError(err).Error("Failed to close session")
		}
	}()

	//set io
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	in, err := session.StdinPipe()

	modes := ssh.TerminalModes{ssh.ECHO: 0, ssh.TTY_OP_ISPEED: 14400, ssh.TTY_OP_OSPEED: 14400}

	if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
		logger.WithError(err).Error("Failed to get pty on dryad.")
		session.Close()
	}
	/*
		stdin, err := session.StdinPipe()
		if err != nil {
			logger.WithError(err).Error("Unable to setup stdin for session.")
		}
		go io.Copy(stdin, os.Stdin)
		stdout, err := session.StdoutPipe()
		if err != nil {
			logger.WithError(err).Error("Unable to setup stdout for session.")
		}
		go io.Copy(os.Stdout, stdout)
		stderr, err := session.StderrPipe()
		if err != nil {
			logger.WithError(err).Error("Unable to setup stderr for session.")
		}
		go io.Copy(os.Stderr, stderr)
		// start remote shell
	*/
	if err := session.Shell(); err != nil {
		logger.WithError(err).Error("failed to start shell on dryad")
		return
	}
	// accepting commands
	for {
		reader := bufio.NewReader(os.Stdin)
		str, _ := reader.ReadString('\n')
		fmt.Fprint(in, str)
	}

	/*
			sshCmdArgs := []string{"-tt",
				"-vvv", //verbose
				"-o StrictHostKeyChecking=no",
				"-o UserKnownHostsFile=/dev/null",
				"-o IdentitiesOnly=yes",
				"-i " + identityFilePath,
				"-p " + port,
				"-l " + accessInfo.Username,
				"106.120.47.240"}

			sshCmd := exec.Command("ssh", sshCmdArgs...)

			fmt.Println("ssh", strings.Join(sshCmdArgs, " "), "\n\n")
		ptmx, err := pty.Start(sshCmd)
		if err != nil {
			logger.WithError(err).Error("Failed to create pseudo terminal for ssh.")
		}
		defer func() {
			if err := ptmx.Close(); err != nil {
				logger.WithError(err).Error("Failed to close pseudo terminal for ssh.")
			}

		}()

		// Handle pty size.
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGWINCH)
		go func() {
			for range ch {
				if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
					logger.WithError(err).Errorf("error resizing pty: %s", err)
				}
			}
		}()
		ch <- syscall.SIGWINCH // Initial resize.

		// Set stdin in raw mode.
		oldState, err := terminal.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			panic(err)
		}
		defer func() { _ = terminal.Restore(int(os.Stdin.Fd()), oldState) }() // Best effort.

		// Copy stdin to the pty and the pty to stdout.
		go func() { _, _ = io.Copy(ptmx, os.Stdin) }()
		_, _ = io.Copy(os.Stdout, ptmx)

		//out, err := sshCmd.CombinedOutput()
		//fmt.Printf("\n%s\n", string(out))

	*/
	// TODO: handle errs below

	/*
		go func() {
			sshStdin, _ := sshCmd.StdinPipe()
			defer sshStdin.Close()
			for {
				io.Copy(sshStdin, os.Stdin)
			}
		}()
		go func() {
			sshStdout, _ := sshCmd.StdoutPipe()
			defer sshStdout.Close()
			for {
				io.Copy(os.Stdout, sshStdout)
			}
		}()
		go func() {
			sshStderr, _ := sshCmd.StderrPipe()
			defer sshStderr.Close()
			// closing stderr migh cause crashes as Go runtime also uses it.
			for {
				io.Copy(os.Stderr, sshStderr)
			}
		}()
	*/
	// err = sshCmd.Run()
	//fmt.Println(err)
	return
}
