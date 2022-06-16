package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
	"rogchap.com/v8go"
)

func Slugify(raw string) string {
	return strings.ReplaceAll(raw, "/", "_")
}

type Route struct {
	Path    string
	Handler func(req *Request, res *Response) error
}

type Bundler struct {
	Pages []*Page
}

func NewBundler() *Bundler {
	return &Bundler{
		Pages: []*Page{},
	}
}

func (b *Bundler) EnqueuePage(page *Page) {
	b.Pages = append(b.Pages, page)
}

func (b *Bundler) BuildViewsServerSide() error {
	os.RemoveAll("./dist/server")
	result := api.Build(api.BuildOptions{
		Stdin: &api.StdinOptions{
			Contents:   `export * from "@nexus/server-build"`,
			ResolveDir: ".",
			Sourcefile: "main.jsx",
		},
		Outfile:           "./dist/server/index.js",
		Write:             true,
		Bundle:            true,
		Platform:          api.PlatformNeutral,
		Format:            api.FormatCommonJS,
		ResolveExtensions: []string{".jsx", ".tsx"},
		External:          []string{"util", "stream"},
		TreeShaking:       api.TreeShakingTrue,
		Plugins: []api.Plugin{
			{
				Name: "NexusServerEntry",
				Setup: func(build api.PluginBuild) {
					filter := `^@nexus\/server-build$`
					build.OnResolve(api.OnResolveOptions{
						Filter: filter,
					}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
						return api.OnResolveResult{
							Path:      args.Path,
							Namespace: "server-entry-module",
						}, nil
					})

					build.OnLoad(api.OnLoadOptions{
						Filter: filter,
					}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
						var contentsBuilder strings.Builder

						contentsBuilder.WriteString("import * as server_entry from \"./app/entry.server.jsx\"\n")
						for _, page := range b.Pages {
							pagePath := path.Join("./app/routes", page.Path, "page.jsx")
							slugifiedPath := Slugify(page.Path)
							if slugifiedPath == "" {
								slugifiedPath = "index"
							}
							pageSlug := fmt.Sprintf("route_%s_page", slugifiedPath)
							contentsBuilder.WriteString(fmt.Sprintf("import * as %s from \"./%s\"\n", pageSlug, pagePath))
						}

						contentsBuilder.WriteString("export const entry = {module:server_entry}\n")
						contentsBuilder.WriteString("export const routes = [\n")
						for _, page := range b.Pages {
							slugifiedPath := Slugify(page.Path)
							if slugifiedPath == "" {
								slugifiedPath = "index"
							}
							pageSlug := fmt.Sprintf("route_%s_page", slugifiedPath)
							contentsBuilder.WriteString(fmt.Sprintf(`{path:"%s",view:%s},`+"\n", page.Path, pageSlug))
						}
						contentsBuilder.WriteString("]\n")

						contents := contentsBuilder.String()
						return api.OnLoadResult{
							ResolveDir: ".",
							Loader:     api.LoaderJS,
							Contents:   &contents,
						}, nil
					})
				},
			},
		},
	})
	if len(result.Errors) > 0 {
		err := result.Errors[0]
		return fmt.Errorf("%s\n%s:%d", err.Text, err.Location.File, err.Location.Line)
	}
	return nil
}

func (b *Bundler) RenderPage(page *Page, data interface{}) (string, error) {
	iso := v8go.NewIsolate()

	streamModuleTemplate := v8go.NewObjectTemplate(iso)
	readableClassTempalte := v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		return nil
	})
	if err := streamModuleTemplate.Set("Readable", readableClassTempalte); err != nil {
		return "", err
	}

	utilModuleTemplate := v8go.NewObjectTemplate(iso)

	textEncoderEncodeTemplate := v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		return nil
	})

	textEncoderClassTemplate := v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		this := info.This()
		textEncoderEncode := textEncoderEncodeTemplate.GetFunction(info.Context())
		if err := this.Set("encode", textEncoderEncode); err != nil {
			panic(err)
		}
		return nil
	})
	if err := utilModuleTemplate.Set("TextEncoder", textEncoderClassTemplate); err != nil {
		return "", err
	}

	globalTemplate := v8go.NewObjectTemplate(iso)
	moduleTemplate := v8go.NewObjectTemplate(iso)
	if err := globalTemplate.Set("module", moduleTemplate); err != nil {
		return "", err
	}
	processTemplate := v8go.NewObjectTemplate(iso)
	processEnvTemplate := v8go.NewObjectTemplate(iso)
	if err := processTemplate.Set("env", processEnvTemplate); err != nil {
		return "", err
	}
	if err := globalTemplate.Set("process", processTemplate); err != nil {
		return "", err
	}
	requireTemplate := v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		moduleName := info.Args()[0].String()
		if moduleName == "stream" {
			streamModule, err := streamModuleTemplate.NewInstance(info.Context())
			if err != nil {
				panic(err)
			}
			return streamModule.Value
		}
		if moduleName == "util" {
			utilModule, err := utilModuleTemplate.NewInstance(info.Context())
			if err != nil {
				panic(err)
			}
			return utilModule.Value
		}
		return nil
	})
	if err := globalTemplate.Set("require", requireTemplate); err != nil {
		return "", err
	}
	responseClassTemplate := v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		body := info.Args()[0].String()
		if err := info.This().Set("body", body); err != nil {
			panic(err)
		}
		return nil
	})
	if err := globalTemplate.Set("Response", responseClassTemplate); err != nil {
		return "", err
	}
	ctx := v8go.NewContext(iso, globalTemplate)
	serverScriptBytes, err := ioutil.ReadFile("./dist/server/index.js")
	if err != nil {
		return "", err
	}
	serverScript := string(serverScriptBytes)
	_, err = ctx.RunScript(serverScript, "<main>")
	if err != nil {
		return "", err
	}
	dataSerialized, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	propsString, err := v8go.NewValue(iso, string(dataSerialized))
	if err != nil {
		return "", err
	}
	if err := ctx.Global().Set("props", propsString); err != nil {
		return "", err
	}
	entryScript := "entry.module.default({routes:routes,props:JSON.parse(props)}, {url:\"\"})"
	response, err := ctx.RunScript(entryScript, "<server>")
	if err != nil {
		return "", err
	}
	body, err := response.Object().Get("body")
	if err != nil {
		return "", err
	}
	return body.String(), nil
}

type Router struct {
	Bundler *Bundler
	Routes  []*Route
}

func NewRouter() *Router {
	return &Router{
		Bundler: NewBundler(),
		Routes:  []*Route{},
	}
}

func (r *Router) Page(page *Page) *Router {
	r.Bundler.EnqueuePage(page)

	r.Routes = append(r.Routes, &Route{
		Path: page.Path,
		Handler: func(req *Request, res *Response) error {
			data, err := page.Loader(req)
			if err != nil {
				return err
			}
			pageHtml, err := r.Bundler.RenderPage(page, data)
			if err != nil {
				return err
			}
			println(pageHtml)
			return nil
		},
	})
	return r
}

type Request struct{}

type Response struct{}

type Page struct {
	Path   string
	Loader func(req *Request) (interface{}, error)
}

type CounterModel struct {
	Count int32 `json:"counter"`
}

func main() {
	r := NewRouter()
	r.Page(&Page{
		Path: "",
		Loader: func(req *Request) (interface{}, error) {
			return &CounterModel{
				Count: 0,
			}, nil
		},
	})
	if err := r.Bundler.BuildViewsServerSide(); err != nil {
		panic(err)
	}
	if err := r.Routes[0].Handler(&Request{}, &Response{}); err != nil {
		panic(err)
	}
}
