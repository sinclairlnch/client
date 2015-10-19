package systests

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/keybase/client/go/client"
	"github.com/keybase/client/go/libkb"
	keybase1 "github.com/keybase/client/go/protocol"
	"github.com/keybase/client/go/service"
	rpc "github.com/keybase/go-framed-msgpack-rpc"
	"io"
	"testing"
)

type signupInfo struct {
	username   string
	email      string
	passphrase string
}

type signupUI struct {
	baseNullUI
	info *signupInfo
	libkb.Contextified
}

type signupTerminalUI struct {
	info *signupInfo
	libkb.Contextified
}

type signupSecretUI struct {
	info *signupInfo
	libkb.Contextified
}

type signupLoginUI struct {
	libkb.Contextified
}

type signupGPGUI struct {
	libkb.Contextified
}

func (n *signupUI) GetTerminalUI() libkb.TerminalUI {
	return &signupTerminalUI{
		info:         n.info,
		Contextified: libkb.NewContextified(n.G()),
	}
}

func (n *signupUI) GetSecretUI() libkb.SecretUI {
	return &signupSecretUI{
		info:         n.info,
		Contextified: libkb.NewContextified(n.G()),
	}
}

func (n *signupUI) GetLoginUI() libkb.LoginUI {
	return client.NewLoginUI(n.GetTerminalUI(), false)
}

func (n *signupUI) GetGPGUI() libkb.GPGUI {
	return client.NewGPGUI(n.GetTerminalUI(), false)
}

func (n *signupSecretUI) GetNewPassphrase(arg keybase1.GetNewPassphraseArg) (res keybase1.GetNewPassphraseRes, err error) {
	res.Passphrase = n.info.passphrase
	n.G().Log.Debug("| GetNewPassphrase: %v -> %v", arg, res)
	return res, err
}

func (n *signupSecretUI) GetKeybasePassphrase(arg keybase1.GetKeybasePassphraseArg) (res string, err error) {
	err = fmt.Errorf("GetKeybasePassphrase unimplemented")
	return res, err
}

func (n *signupSecretUI) GetPaperKeyPassphrase(arg keybase1.GetPaperKeyPassphraseArg) (res string, err error) {
	err = fmt.Errorf("GetPaperKeyPassphrase unimplemented")
	return res, err
}

func (n *signupSecretUI) GetSecret(pinentry keybase1.SecretEntryArg, terminal *keybase1.SecretEntryArg) (res *keybase1.SecretEntryRes, err error) {
	err = fmt.Errorf("GetSecret unimplemented")
	return res, err
}

func (n *signupTerminalUI) Prompt(pd libkb.PromptDescriptor, s string) (ret string, err error) {
	switch pd {
	case client.PromptDescriptorSignupUsername:
		ret = n.info.username
	case client.PromptDescriptorSignupEmail:
		ret = n.info.email
	case client.PromptDescriptorSignupDevice:
		ret = "work computer"
	case client.PromptDescriptorSignupCode:
		ret = "202020202020202020202020"
	default:
		err = fmt.Errorf("unknown prompt %v", pd)
	}
	n.G().Log.Debug("Terminal Prompt %d: %s -> %s (%v)\n", pd, s, ret, libkb.ErrToOk(err))
	return ret, err
}

func (n *signupTerminalUI) PromptPassword(pd libkb.PromptDescriptor, _ string) (string, error) {
	return "", nil
}
func (n *signupTerminalUI) Output(s string) error {
	n.G().Log.Debug("Terminal Output: %s", s)
	return nil
}
func (n *signupTerminalUI) Printf(f string, args ...interface{}) (int, error) {
	s := fmt.Sprintf(f, args...)
	n.G().Log.Debug("Terminal Printf: %s", s)
	return len(s), nil
}

func (n *signupTerminalUI) Write(b []byte) (int, error) {
	n.G().Log.Debug("Terminal write: %s", string(b))
	return len(b), nil
}

func (n *signupTerminalUI) OutputWriter() io.Writer {
	return n
}

func (n *signupTerminalUI) PromptYesNo(pd libkb.PromptDescriptor, s string, def libkb.PromptDefault) (ret bool, err error) {
	switch pd {
	case client.PromptDescriptorLoginWritePaper:
		ret = true
	case client.PromptDescriptorLoginWallet:
		ret = true
	case client.PromptDescriptorGPGOKToAdd:
		ret = false
	default:
		err = fmt.Errorf("unknown prompt %v", pd)
	}
	n.G().Log.Debug("Terminal PromptYesNo %d: %s -> %s (%v)\n", pd, s, ret, libkb.ErrToOk(err))
	return ret, err
}

func randomUser(prefix string) *signupInfo {
	b := make([]byte, 5)
	rand.Read(b)
	sffx := hex.EncodeToString(b)
	username := fmt.Sprintf("%s_%s", prefix, sffx)
	return &signupInfo{
		username:   username,
		email:      username + "@noemail.keybase.io",
		passphrase: sffx,
	}
}

type notifyHandler struct {
	logoutCh chan struct{}
	errCh    chan error
}

func newNotifyHandler() *notifyHandler {
	return &notifyHandler{
		logoutCh: make(chan struct{}),
		errCh:    make(chan error),
	}
}

func (h *notifyHandler) LoggedOut() error {
	h.logoutCh <- struct{}{}
	return nil
}

func TestSignupLogout(t *testing.T) {
	tc := setupTest(t, "signup")

	defer tc.Cleanup()

	stopCh := make(chan error)
	svc := service.NewService(false, tc.G)
	startCh := svc.GetStartChannel()
	go func() {
		stopCh <- svc.Run()
	}()

	tc2 := cloneContext(tc)
	sui := signupUI{
		info:         randomUser("sgnp"),
		Contextified: libkb.NewContextified(tc2.G),
	}
	tc2.G.SetUI(&sui)
	signup := client.NewCmdSignupRunner(tc2.G)
	signup.SetTest()

	tc4 := cloneContext(tc)
	logout := client.NewCmdLogoutRunner(tc4.G)

	tc3 := cloneContext(tc)
	stopper := client.NewCmdCtlStopRunner(tc3.G)

	<-startCh

	if err := signup.Run(); err != nil {
		t.Fatal(err)
	}

	nh := newNotifyHandler()

	// Launch the server that will listen for notifications on updates, such as logout
	launchServer := func(nh *notifyHandler) error {
		tc5 := cloneContext(tc)
		cli, xp, err := client.GetRPCClientWithContext(tc5.G)
		if err != nil {
			return err
		}
		srv := rpc.NewServer(xp, nil)
		if err = srv.Register(keybase1.NotifySessionProtocol(nh)); err != nil {
			return err
		}
		ncli := keybase1.NotifyCtlClient{Cli: cli}
		if err = ncli.ToggleNotifications(keybase1.NotificationChannels{Session: true}); err != nil {
			return err
		}
		return nil
	}

	// Actually launch it in the background
	go func() {
		err := launchServer(nh)
		if err != nil {
			nh.errCh <- err
		}
	}()

	// Fire a logout
	if err := logout.Run(); err != nil {
		t.Fatal(err)
	}

	// Now let's be sure that we get a notification back as we expect.
	select {
	case err := <-nh.errCh:
		t.Fatalf("Error before notify: %v", err)
	case <-nh.logoutCh:
		tc.G.Log.Debug("Got notification from logout handler")
	}

	if err := stopper.Run(); err != nil {
		t.Fatal(err)
	}

	// If the server failed, it's also an error
	if err := <-stopCh; err != nil {
		t.Fatal(err)
	}
}
