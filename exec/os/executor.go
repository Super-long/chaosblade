/*
 * Copyright 1999-2020 Alibaba Group Holding Ltd.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package os

import (
	"context"
	"fmt"
	"github.com/chaosblade-io/chaosblade-exec-os/exec"
	"github.com/chaosblade-io/chaosblade-spec-go/log"
	"github.com/chaosblade-io/chaosblade-spec-go/spec"
	"github.com/chaosblade-io/chaosblade-spec-go/util"
	os_exec "os/exec"
	"path"
	"strings"
	"syscall"
)

type Executor struct {
}

func NewExecutor() spec.Executor {
	return &Executor{}
}

func (*Executor) Name() string {
	return "os"
}

func (e *Executor) Exec(uid string, ctx context.Context, model *spec.ExpModel) *spec.Response {

	if model.ActionFlags[exec.ChannelFlag.Name] == "ssh" {
		sshExecutor := &exec.SSHExecutor{}
		return sshExecutor.Exec(uid, ctx, model)
	}

	var args string
	var flags string
	for k, v := range model.ActionFlags {
		if v == "" ||  k == "timeout" {
			continue
		}
		flags = fmt.Sprintf("%s --%s=%s", flags, k, v)
	}
	// 通过withvalue修改的
	_, isDestroy := spec.IsDestroy(ctx)

	if isDestroy {
		args = fmt.Sprintf("%s %s %s%s uid=%s", spec.Destroy, model.Target, model.ActionName, flags, uid)
	} else {
		args = fmt.Sprintf("%s %s %s%s uid=%s", spec.Create, model.Target, model.ActionName, flags, uid)
	}
	chaosOsBin := path.Join(util.GetProgramPath(), "bin", spec.ChaosOsBin)
	argsArray := strings.Split(args, " ")
	command := os_exec.CommandContext(ctx, chaosOsBin, argsArray...)
	log.Debugf(ctx, "run command, %s %s", chaosOsBin, args)

	if model.ActionProcessHang && !isDestroy {
		// 和run的区别是start不等待其执行完成
		if err := command.Start(); err != nil {
			sprintf := fmt.Sprintf("create experiment command start failed, %v", err)
			return spec.ReturnFail(spec.OsCmdExecFailed, sprintf)
		}
		command.SysProcAttr = &syscall.SysProcAttr{}
		return spec.ReturnSuccess(command.Process.Pid)
	} else {
		// 先Run然后再打印标准输出和标准错误
		output, err := command.CombinedOutput()
		outMsg := string(output)
		log.Debugf(ctx, "Command Result, output: %v, err: %v", outMsg, err)
		if err != nil {
			return spec.ReturnFail(spec.OsCmdExecFailed, fmt.Sprintf("command exec failed, %s", err.Error()))
		}
		return spec.Decode(outMsg, nil)
	}
}

func (*Executor) SetChannel(channel spec.Channel) {
}
