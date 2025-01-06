package cmd

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/edgewize-io/image-packaging-tool/pkg/constants"
	"github.com/edgewize-io/image-packaging-tool/pkg/imageref"
	"github.com/edgewize-io/image-packaging-tool/pkg/lock"
	"github.com/edgewize-io/image-packaging-tool/pkg/regctl/archive"
	"github.com/edgewize-io/image-packaging-tool/pkg/server"
	"github.com/edgewize-io/image-packaging-tool/pkg/utils"
	"github.com/regclient/regclient/mod"
	"github.com/regclient/regclient/types/platform"
	"github.com/regclient/regclient/types/ref"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type BuildOptions struct {
	rootOpts             *RootOptions
	baseImage            string
	targetImage          string
	platforms            string
	skipScript           bool
	outputServerFilePath string
	deviceType           string
}

type ModelTemplateParam struct {
	Name       string
	DeviceType string
}

func NewCmdBuild(rootOptions *RootOptions) *cobra.Command {
	buildOptions := &BuildOptions{
		rootOpts: rootOptions,
	}

	command := &cobra.Command{
		Use:   "build",
		Short: "build model image",
		Long:  "init new model for current image",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				buildOptions.targetImage = args[0]
			} else {
				return fmt.Errorf("target image name cannot be empty")
			}

			err := buildOptions.validate()
			if err != nil {
				return err
			}

			return buildOptions.build()
		},
	}

	flags := command.Flags()
	flags.StringVar(&buildOptions.baseImage, "baseImage", "", "base infer model image")
	flags.StringVar(&buildOptions.platforms, "platforms", "linux/amd64", "build image platforms, default: linux/amd64")
	flags.BoolVar(&buildOptions.skipScript, "skipScript", false, "skip generating serving_server.py, default: false")
	flags.StringVar(&buildOptions.outputServerFilePath, "file", "server.yaml", "output server.yaml path")
	flags.StringVar(&buildOptions.deviceType, "deviceType", "Ascend", "device type, support: [\"CPU\", \"GPU\", \"Ascend\"], default \"Ascend\"")

	return command
}

func (bo *BuildOptions) validate() error {
	switch strings.ToLower(bo.deviceType) {
	default:
		return fmt.Errorf("unknown deviceType [%s]", bo.deviceType)
	case "cpu":
		utils.PrintWarning(os.Stdout, fmt.Sprintf("deviceType is CPU now\n"))
	case "ascend", "gpu":
	}

	if bo.baseImage == "" {
		return fmt.Errorf("base image cannnot be empty!")
	}

	if bo.skipScript {
		utils.PrintYellow(os.Stdout, fmt.Sprintf("please provide serving_server.py yourself\n"))
	}

	return nil
}

func (bo *BuildOptions) build() (err error) {
	imageRef, err := imageref.NewImageRef(bo.targetImage)
	if err != nil {
		utils.PrintWarning(os.Stdout, fmt.Sprintf("targetImage [%s] reference invalid\n", bo.targetImage))
		return
	}

	ctx := context.TODO()
	currWorkDir, err := os.Getwd()
	if err != nil {
		return
	}

	lockFilePath := filepath.Join(currWorkDir, constants.MetaDirName, constants.LockFileName)
	workspaceLocked := lock.LockFile(lockFilePath)
	if !workspaceLocked {
		utils.PrintWarning(os.Stdout, fmt.Sprintf("\ntry to lock workspace failed, please wait or clean workspace\n"))
		return
	}

	defer lock.UnlockFile(lockFilePath)

	if !bo.skipScript {
		err = bo.renderServingServer(currWorkDir)
		if err != nil {
			utils.PrintWarning(os.Stdout, fmt.Sprintf("generate serving_server.py failed, err: %v\n", err))
			return
		}

		utils.PrintString(os.Stdout, fmt.Sprintf("serving_server.py generated successfully!\n"))
	} else {
		fileExist := bo.checkServingServerFile(currWorkDir)
		if !fileExist {
			utils.PrintWarning(os.Stdout, fmt.Sprintf("must provide serving_server.py yourself!\n"))
			return
		}
	}

	startScriptPath, err := bo.createStartScript(currWorkDir)
	if err != nil {
		utils.PrintWarning(os.Stdout, fmt.Sprintf("generate image started script failed, err: %v\n", err))
		return
	}

	var rdr io.Reader
	var platforms []platform.Platform

	pf, err := platform.Parse(bo.platforms)
	if err != nil {
		err = fmt.Errorf("failed to parse platform %s: %v", bo.platforms, err)
		return
	}

	platforms = append(platforms, pf)

	pr, pw := io.Pipe()
	go func() {
		err := archive.Tar(context.TODO(), currWorkDir, pw)
		if err != nil {
			_ = pw.CloseWithError(err)
		}
		_ = pw.Close()
	}()
	rdr = pr
	defer pr.Close()

	rSrc, err := ref.New(bo.baseImage)
	if err != nil {
		return
	}

	var rTgt ref.Ref
	if strings.ContainsAny(bo.targetImage, "/:") {
		rTgt, err = ref.New(bo.targetImage)
		if err != nil {
			err = fmt.Errorf("failed to parse new image name %s: %w", bo.targetImage, err)
			return
		}
	} else {
		rTgt = rSrc.SetTag(bo.targetImage)
	}

	modOptions := []mod.Opts{}

	modOptions = append(modOptions,
		mod.WithLayerAddTar(rdr, "", platforms),
		mod.WithRefTgt(rTgt),
		mod.WithConfigEntrypoint([]string{"/bin/bash", "-c", startScriptPath}),
	)

	rc := bo.rootOpts.newRegClient()
	defer rc.Close(ctx, rSrc)
	rOut, err := mod.Apply(ctx, rc, rSrc, modOptions...)
	if err != nil {
		return
	}

	utils.PrintString(os.Stdout, fmt.Sprintf("image %s pushed to registry successfully\n", rOut.CommonName()))
	err = rc.Close(ctx, rOut)
	if err != nil {
		err = fmt.Errorf("failed to close ref: %w", err)
	}

	err = bo.updateImageInfo(imageRef)
	if err != nil {
		utils.PrintWarning(os.Stdout, fmt.Sprintf("export server.yaml failed\n", rOut.CommonName()))
	}
	return
}

func (bo *BuildOptions) updateImageInfo(imageRef imageref.ImageRef) (err error) {
	currWorkDir, err := os.Getwd()
	if err != nil {
		return
	}

	metaDirPath := filepath.Join(currWorkDir, constants.MetaDirName)
	_, err = os.Stat(metaDirPath)
	if err != nil {
		utils.PrintWarning(os.Stdout, fmt.Sprintf(".modelmesh not found, current workspace not initialized correctly\n"))
		return
	}

	serverConfigFilePath := filepath.Join(metaDirPath, constants.ServerConfigFile)
	serverConfigBytes, err := os.ReadFile(serverConfigFilePath)
	if err != nil {
		utils.PrintWarning(os.Stdout, fmt.Sprintf("read current server.yaml content failed, %v\n", err))
		return
	}

	serverConfig := &server.ServerFile{}
	err = yaml.Unmarshal(serverConfigBytes, serverConfig)
	if err != nil {
		return
	}

	subModelDirs, err := bo.getSubDirectories(currWorkDir)
	if err != nil {
		return
	}

	newServables := []server.ServableConfig{}
	for _, subModelName := range subModelDirs {
		servableConfig := server.ServableConfig{
			Name:        subModelName,
			Description: fmt.Sprintf("infer model %s", subModelName),
		}

		methodDetails, _err := GetModelMethods(filepath.Join(currWorkDir, subModelName))
		if _err != nil {
			utils.PrintYellow(os.Stdout, fmt.Sprintf("get methods Docs for model directory [%s] failed\n", subModelName))
			continue
		}

		servableConfig.Methods = methodDetails
		newServables = append(newServables, servableConfig)
	}

	serverConfig.Servables = newServables

	serverConfig.Image = server.ImageInfo{
		Registry:   imageRef.Registry,
		Repository: imageRef.Repository,
		Tag:        imageRef.Tag,
	}

	updatedServerConfigBytes, err := yaml.Marshal(serverConfig)
	if err != nil {
		return
	}

	err = os.WriteFile(serverConfigFilePath, updatedServerConfigBytes, 0666)
	if err != nil {
		return
	}

	err = CopyFile(serverConfigFilePath, bo.outputServerFilePath)
	if err != nil {
		return
	}

	utils.PrintString(os.Stdout, fmt.Sprintf("export server.yaml successffully\n"))
	return
}

func (bo *BuildOptions) checkServingServerFile(currWorkDir string) bool {
	_, err := os.Stat(filepath.Join(currWorkDir, constants.ServingServerFile))
	if err != nil {
		return false
	}

	return true
}

func (bo *BuildOptions) renderServingServer(currWorkDir string) (err error) {
	templateParams, err := bo.getTemplateParams(currWorkDir)
	if err != nil {
		return
	}

	const fileTemplate = `
import os
import sys
from mindspore_serving import server

def start():
    servable_dir = os.path.dirname(os.path.realpath(sys.argv[0]))

    servable_config_list = []
    device_ids=0

    # Total 4 worker, one worker occupy device 0, the model inference tasks of other workers are forwarded to the worker
    # that occupies the device.
    {{ range . }}
    {{.Name}}_config = server.ServableStartConfig(servable_directory=servable_dir, servable_name="{{.Name}}", device_ids=device_ids, device_type="{{.DeviceType}}")
    servable_config_list.append({{.Name}}_config)
    {{ end }}

    server.start_servables(servable_configs=servable_config_list)

    server.start_grpc_server("127.0.0.1:5500")

if __name__ == "__main__":
    start()
`

	t, err := template.New("tmpl").Parse(fileTemplate)
	if err != nil {
		return
	}

	outputFile, err := os.Create(filepath.Join(currWorkDir, constants.ServingServerFile))
	if err != nil {
		return
	}

	defer outputFile.Close()

	err = t.Execute(outputFile, templateParams)
	return
}

func (bo *BuildOptions) createStartScript(currWorkDir string) (scriptPath string, err error) {
	const scriptTemplate = `#!/bin/bash
source /usr/local/Ascend/ascend-toolkit/set_env.sh
export LD_LIBRARY_PATH=/usr/local/python3.7.5/lib/python3.7/site-packages/mindspore/lib/:${LD_LIBRARY_PATH}

export PROTOCOL_BUFFERS_PYTHON_IMPLEMENTATION=python
python {{ . }}
`
	t, err := template.New("tmpl").Parse(scriptTemplate)
	if err != nil {
		return
	}

	scriptPath = filepath.Join(currWorkDir, constants.ServingStartScript)
	outputFile, err := os.Create(scriptPath)
	if err != nil {
		return
	}

	defer outputFile.Close()

	err = t.Execute(outputFile, filepath.Join(currWorkDir, constants.ServingServerFile))
	if err != nil {
		return
	}

	err = os.Chmod(scriptPath, 0755)
	return
}

func (bo *BuildOptions) getTemplateParams(currWorkDir string) (params []ModelTemplateParam, err error) {
	subModelDirs, err := bo.getSubDirectories(currWorkDir)
	if err != nil {
		return
	}

	params = []ModelTemplateParam{}
	for _, modelName := range subModelDirs {
		params = append(params, ModelTemplateParam{
			Name:       modelName,
			DeviceType: bo.deviceType,
		})
	}

	return
}

func (bo *BuildOptions) getSubDirectories(currWorkDir string) (subDirs []string, err error) {
	subDirs = []string{}
	dirEntries, err := os.ReadDir(currWorkDir)
	if err != nil {
		return
	}

	for _, dirEntry := range dirEntries {
		if dirEntry.IsDir() && !strings.HasPrefix(dirEntry.Name(), ".") {
			subDirs = append(subDirs, dirEntry.Name())
		}
	}

	return
}

func GetModelMethods(modelDir string) (methods []server.MethodDetail, err error) {
	methods = []server.MethodDetail{}

	entries, err := os.ReadDir(modelDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !strings.HasPrefix(entry.Name(), constants.MethodPrefix) {
			continue
		}

		methodFilePath := filepath.Join(modelDir, entry.Name())

		content, _err := os.ReadFile(methodFilePath)
		if _err != nil {
			utils.PrintYellow(os.Stdout, fmt.Sprintf("read method file [%s] failed\n", methodFilePath))
			continue
		}

		currMethodName := strings.TrimPrefix(RemoveFileExtension(entry.Name()), constants.MethodPrefix)
		methods = append(methods, server.MethodDetail{
			Name:        currMethodName,
			Description: fmt.Sprintf("about how to access method %s", currMethodName),
			Readme:      base64.StdEncoding.EncodeToString(content),
		})
	}

	return
}

func RemoveFileExtension(filename string) string {
	ext := filepath.Ext(filename)
	if ext != "" {
		return strings.TrimSuffix(filename, ext)
	}
	return filename
}

func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	dstFileInfo, err := os.Stat(dst)
	if err == nil && dstFileInfo.IsDir() {
		dst = filepath.Join(dst, filepath.Base(src))
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	utils.PrintString(os.Stdout, fmt.Sprintf("export server.yaml to %s successffully\n", dst))
	return nil
}
