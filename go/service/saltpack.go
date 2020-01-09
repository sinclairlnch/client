// Copyright 2015 Keybase, Inc. All rights reserved. Use of
// this source code is governed by the included BSD license.

package service

import (
	"bytes"
	"errors"
	"io/ioutil"
	"strings"

	"github.com/keybase/client/go/engine"
	"github.com/keybase/client/go/libkb"
	keybase1 "github.com/keybase/client/go/protocol/keybase1"
	"github.com/keybase/client/go/saltpackkeys"
	"github.com/keybase/go-framed-msgpack-rpc/rpc"
	"golang.org/x/net/context"
)

type SaltpackHandler struct {
	*BaseHandler
	libkb.Contextified
}

type RemoteSaltpackUI struct {
	sessionID int
	cli       keybase1.SaltpackUiClient
}

func NewRemoteSaltpackUI(sessionID int, c *rpc.Client) *RemoteSaltpackUI {
	return &RemoteSaltpackUI{
		sessionID: sessionID,
		cli:       keybase1.SaltpackUiClient{Cli: c},
	}
}

func (r *RemoteSaltpackUI) SaltpackPromptForDecrypt(ctx context.Context, arg keybase1.SaltpackPromptForDecryptArg, usedDelegateUI bool) (err error) {
	arg.SessionID = r.sessionID
	arg.UsedDelegateUI = usedDelegateUI
	return r.cli.SaltpackPromptForDecrypt(ctx, arg)
}

func (r *RemoteSaltpackUI) SaltpackVerifySuccess(ctx context.Context, arg keybase1.SaltpackVerifySuccessArg) (err error) {
	arg.SessionID = r.sessionID
	return r.cli.SaltpackVerifySuccess(ctx, arg)
}

func (r *RemoteSaltpackUI) SaltpackVerifyBadSender(ctx context.Context, arg keybase1.SaltpackVerifyBadSenderArg) (err error) {
	arg.SessionID = r.sessionID
	return r.cli.SaltpackVerifyBadSender(ctx, arg)
}

func NewSaltpackHandler(xp rpc.Transporter, g *libkb.GlobalContext) *SaltpackHandler {
	return &SaltpackHandler{
		BaseHandler:  NewBaseHandler(g, xp),
		Contextified: libkb.NewContextified(g),
	}
}

func (h *SaltpackHandler) SaltpackDecrypt(ctx context.Context, arg keybase1.SaltpackDecryptArg) (info keybase1.SaltpackEncryptedMessageInfo, err error) {
	ctx = libkb.WithLogTag(ctx, "SP")
	cli := h.getStreamUICli()
	src := libkb.NewRemoteStreamBuffered(arg.Source, cli, arg.SessionID)
	snk := libkb.NewRemoteStreamBuffered(arg.Sink, cli, arg.SessionID)
	earg := &engine.SaltpackDecryptArg{
		Sink:   snk,
		Source: src,
		Opts:   arg.Opts,
	}

	uis := libkb.UIs{
		IdentifyUI: h.NewRemoteIdentifyUI(arg.SessionID, h.G()),
		SecretUI:   h.getSecretUI(arg.SessionID, h.G()),
		SaltpackUI: h.getSaltpackUI(arg.SessionID),
		SessionID:  arg.SessionID,
	}
	m := libkb.NewMetaContext(ctx, h.G()).WithUIs(uis)
	resolver := saltpackkeys.NewKeyPseudonymResolver(m)
	eng := engine.NewSaltpackDecrypt(earg, resolver)
	err = engine.RunEngine2(m, eng)
	info = eng.MessageInfo()
	return info, err
}

func (h *SaltpackHandler) SaltpackEncrypt(ctx context.Context, arg keybase1.SaltpackEncryptArg) error {
	ctx = libkb.WithLogTag(ctx, "SP")
	cli := h.getStreamUICli()
	src := libkb.NewRemoteStreamBuffered(arg.Source, cli, arg.SessionID)
	snk := libkb.NewRemoteStreamBuffered(arg.Sink, cli, arg.SessionID)
	earg := &engine.SaltpackEncryptArg{
		Opts:   arg.Opts,
		Sink:   snk,
		Source: src,
	}

	uis := libkb.UIs{
		IdentifyUI: h.NewRemoteIdentifyUI(arg.SessionID, h.G()),
		SecretUI:   h.getSecretUI(arg.SessionID, h.G()),
		SessionID:  arg.SessionID,
	}

	keyfinderHook := saltpackkeys.NewRecipientKeyfinderEngineHook(arg.Opts.UseKBFSKeysOnlyForTesting)

	eng := engine.NewSaltpackEncrypt(earg, keyfinderHook)
	m := libkb.NewMetaContext(ctx, h.G()).WithUIs(uis)
	return engine.RunEngine2(m, eng)
}

func (h *SaltpackHandler) SaltpackSign(ctx context.Context, arg keybase1.SaltpackSignArg) error {
	ctx = libkb.WithLogTag(ctx, "SP")
	cli := h.getStreamUICli()
	src := libkb.NewRemoteStreamBuffered(arg.Source, cli, arg.SessionID)
	snk := libkb.NewRemoteStreamBuffered(arg.Sink, cli, arg.SessionID)
	earg := &engine.SaltpackSignArg{
		Opts:   arg.Opts,
		Sink:   snk,
		Source: src,
	}

	uis := libkb.UIs{
		IdentifyUI: h.NewRemoteIdentifyUI(arg.SessionID, h.G()),
		SecretUI:   h.getSecretUI(arg.SessionID, h.G()),
		SessionID:  arg.SessionID,
	}
	m := libkb.NewMetaContext(ctx, h.G()).WithUIs(uis)
	eng := engine.NewSaltpackSign(h.G(), earg)
	return engine.RunEngine2(m, eng)
}

func (h *SaltpackHandler) SaltpackVerify(ctx context.Context, arg keybase1.SaltpackVerifyArg) error {
	ctx = libkb.WithLogTag(ctx, "SP")
	cli := h.getStreamUICli()
	src := libkb.NewRemoteStreamBuffered(arg.Source, cli, arg.SessionID)
	snk := libkb.NewRemoteStreamBuffered(arg.Sink, cli, arg.SessionID)
	earg := &engine.SaltpackVerifyArg{
		Opts:   arg.Opts,
		Sink:   snk,
		Source: src,
	}

	uis := libkb.UIs{
		IdentifyUI: h.NewRemoteIdentifyUI(arg.SessionID, h.G()),
		SecretUI:   h.getSecretUI(arg.SessionID, h.G()),
		SaltpackUI: h.getSaltpackUI(arg.SessionID),
		SessionID:  arg.SessionID,
	}
	eng := engine.NewSaltpackVerify(h.G(), earg)
	m := libkb.NewMetaContext(ctx, h.G()).WithUIs(uis)
	return engine.RunEngine2(m, eng)
}

// frontend handlers:

func (h *SaltpackHandler) SaltpackEncryptString(ctx context.Context, arg keybase1.SaltpackEncryptStringArg) (string, error) {
	ctx = libkb.WithLogTag(ctx, "SP")

	auth := keybase1.AuthenticityType_REPUDIABLE
	if arg.Opts.Signed {
		auth = keybase1.AuthenticityType_SIGNED
	}
	opts := keybase1.SaltpackEncryptOptions{
		Recipients:       arg.Opts.Recipients,
		AuthenticityType: auth,
		NoSelfEncrypt:    !arg.Opts.IncludeSelf,
		UseEntityKeys:    true,
	}
	sink := libkb.NewBufferCloser()
	earg := &engine.SaltpackEncryptArg{
		Opts:   opts,
		Sink:   sink,
		Source: strings.NewReader(arg.Plaintext),
	}

	uis := libkb.UIs{
		SecretUI:  h.getSecretUI(arg.SessionID, h.G()),
		SessionID: arg.SessionID,
	}

	keyfinderHook := saltpackkeys.NewRecipientKeyfinderEngineHook(false)

	eng := engine.NewSaltpackEncrypt(earg, keyfinderHook)
	m := libkb.NewMetaContext(ctx, h.G()).WithUIs(uis)
	err := engine.RunEngine2(m, eng)
	if err != nil {
		return "", err
	}

	return sink.String(), nil
}

func (h *SaltpackHandler) SaltpackDecryptString(ctx context.Context, arg keybase1.SaltpackDecryptStringArg) (keybase1.SaltpackPlaintextResult, error) {
	ctx = libkb.WithLogTag(ctx, "SP")
	return keybase1.SaltpackPlaintextResult{}, errors.New("nyi")
}

func (h *SaltpackHandler) SaltpackSignString(ctx context.Context, arg keybase1.SaltpackSignStringArg) (string, error) {
	ctx = libkb.WithLogTag(ctx, "SP")
	sink := libkb.NewBufferCloser()
	earg := &engine.SaltpackSignArg{
		Sink:   sink,
		Source: ioutil.NopCloser(bytes.NewBufferString(arg.Plaintext)),
	}

	uis := libkb.UIs{
		SecretUI:  h.getSecretUI(arg.SessionID, h.G()),
		SessionID: arg.SessionID,
	}
	m := libkb.NewMetaContext(ctx, h.G()).WithUIs(uis)
	eng := engine.NewSaltpackSign(h.G(), earg)
	if err := engine.RunEngine2(m, eng); err != nil {
		return "", err
	}

	return sink.String(), nil
}

func (h *SaltpackHandler) SaltpackVerifyString(ctx context.Context, arg keybase1.SaltpackVerifyStringArg) (keybase1.SaltpackVerifyResult, error) {
	ctx = libkb.WithLogTag(ctx, "SP")
	return keybase1.SaltpackVerifyResult{}, errors.New("nyi")
}
