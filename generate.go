package main

import (
	"fmt"
	"unicode"

	"google.golang.org/protobuf/compiler/protogen"
)

func generateHTTPServers(
	srvs []Server,
	g *protogen.GeneratedFile,
) error {

	// imports
	g.P("import (")
	g.P("\t\"context\"")
	g.P("\t\"github.com/gin-gonic/gin\"")
	g.P("\"github.com/mailru/easyjson\"")
	g.P(")")

	for _, srv := range srvs {
		intname := srv.Service.GoName + "HTTPServer"
		g.P(fmt.Sprintf("// %s", srv.Service.GoName))
		g.P("type ", intname, " interface {")
		for _, rpc := range srv.Paths {
			g.Write([]byte(rpc.Method.Comments.Leading.String()))
			g.P("\t", rpc.Method.GoName, "(context.Context, *", rpc.Method.Input.GoIdent.GoName, ") (*", rpc.Method.Output.GoIdent.GoName, ", error)")
			g.Write([]byte(rpc.Method.Comments.Trailing.String()))
		}
		g.P("}")

		// controllers
		// TODO: handle path and query parameter type :)
		ctrlName := toPrivateName(srv.Service.GoName)
		g.P("type ", ctrlName, " struct {")
		g.P("app ", intname)
		g.P("}")

		for _, rpc := range srv.Paths {

			g.P("// ", rpc.Description)
			g.P("func (p *", ctrlName, ")", toPrivateName(rpc.Method.GoName), "(ctx *gin.Context) {")

			g.P("body := ", rpc.Method.Input.GoIdent.GoName, "{}")
			if rpc.HTTPMethod != "GET" {
				// if anything left in body
				g.P("easyjson.UnmarshalFromReader(ctx.Request.Body, &body)")
			}
			for _, qpm := range rpc.QueryParameters {
				g.P("body.", qpm.ModelParameter, "= ctx.Query(\",", qpm.Key, "\")")
			}
			for _, pth := range rpc.PathParameters {
				g.P("body.", pth.ModelParameter, "= ctx.Param(\",", pth.Key, "\")")
			}

			g.P("p.app.", rpc.Method.GoName, "(")
			g.P("ctx,")
			g.P("&body,")
			g.P(")")
			g.P("}")
		}

		g.P("func Register", srv.Service.GoName, "HTTPServer (")
		g.P("grp *gin.RouterGroup,")
		g.P("srv ", intname, ",")
		g.P(") {")
		g.P("ctrl := ", ctrlName, "{app: srv}")
		for _, rpc := range srv.Paths {
			g.P("grp.", rpc.HTTPMethod, "(\"", rpc.Path, "\", ", "ctrl.", toPrivateName(rpc.Method.GoName), ")")
		}
		g.P("}")
	}

	return nil
}

func generateOpenAPI(
	srvs []Server,
	g *protogen.GeneratedFile,
	file *protogen.File,
) error {
	g.P("openapi: 3.0.3")
	g.P("info:")
	g.P("  title: ", file.Desc.Package())
	return nil
}

func toPrivateName(in string) (out string) {
	inr := []rune(in)
	inr[0] = unicode.ToLower(inr[0])
	out = string(inr)
	return
}
