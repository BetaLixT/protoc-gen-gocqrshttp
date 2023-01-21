package pkg

import (
	"fmt"
	"strings"
	"unicode"

	"google.golang.org/protobuf/compiler/protogen"
)

func GenerateHTTPServers(
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
		ctrlName := ToPrivateName(srv.Service.GoName)
		g.P("type ", ctrlName, " struct {")
		g.P("app ", intname)
		g.P("}")

		for _, rpc := range srv.Paths {

			g.P("// ", rpc.Description)
			g.P("func (p *", ctrlName, ")", ToPrivateName(rpc.Method.GoName), "(ctx *gin.Context) {")

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
			g.P("grp.", rpc.HTTPMethod, "(\"", rpc.Path, "\", ", "ctrl.", ToPrivateName(rpc.Method.GoName), ")")
		}
		g.P("}")
	}

	return nil
}

func GenerateOpenAPI(
	srvs []Server,
	g *protogen.GeneratedFile,
	file *protogen.File,
) error {
	g.P("openapi: 3.0.3")
	g.P("info:")
	g.P("  title: ", file.Desc.Package())
	g.P("paths:")
	for _, svc := range srvs {
		for _, api := range svc.Paths {
			g.P("  ", api.Path, ":")
			g.P("    ", strings.ToLower(api.HTTPMethod), ":")
			if len(api.Tags) != 0 {
				g.P("      tags:")
				for _, tag := range api.Tags {
					g.P("        - ", tag)
				}
			}
			g.P("      summary: ", api.Summary)         // TODO: escaping
			g.P("      description: ", api.Description) // TODO: escaping
			// g.P("      requestBody: ")

		}
		// paths:
		//   /pet:
		//     put:
		//       tags:
		//         - pet
		//       summary: Update an existing pet
		//       description: Update an existing pet by Id
		//       operationId: updatePet
		//       requestBody:
		//         description: Update an existent pet in the store
		//         content:
		//           application/json:
		//             schema:
		//               $ref: '#/components/schemas/Pet'
		//           application/xml:
		//             schema:
		//               $ref: '#/components/schemas/Pet'
		//           application/x-www-form-urlencoded:
		//             schema:
		//               $ref: '#/components/schemas/Pet'
		//         required: true
		//       responses:
		//         '200':
		//           description: Successful operation
		//           content:
		//             application/json:
		//               schema:
		//                 $ref: '#/components/schemas/Pet'
		//             application/xml:
		//               schema:
		//                 $ref: '#/components/schemas/Pet'
		//         '400':
		//           description: Invalid ID supplied
		//         '404':
		//           description: Pet not found
		//         '405':
		//           description: Validation exception
		//       security:
		//         - petstore_auth:
		//             - write:pets
		//             - read:pets
	}
	return nil
}

func ToPrivateName(in string) (out string) {
	inr := []rune(in)
	inr[0] = unicode.ToLower(inr[0])
	out = string(inr)
	return
}
