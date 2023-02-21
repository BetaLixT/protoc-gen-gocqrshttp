// Package pkg package
package pkg

import (
	"fmt"
	"strings"
	"unicode"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// GenerateHTTPServers generates http servers
func GenerateHTTPServers(
	srvs []Server,
	g *protogen.GeneratedFile,
	file *protogen.File,
) error {
	contextPackage := protogen.GoImportPath("context")
	ginPackage := protogen.GoImportPath("github.com/gin-gonic/gin")
	protojsonPackage := protogen.GoImportPath("google.golang.org/protobuf/encoding/protojson")
	ioutilPackage := protogen.GoImportPath("io/ioutil")

	for _, srv := range srvs {
		intname := srv.Service.GoName + "HTTPServer"
		g.P(fmt.Sprintf("// %s", srv.Service.GoName))
		g.P("type ", intname, " interface {")
		for _, rpc := range srv.Paths {
			// inputStructName := rpc.Method.Input.GoIdent.GoName
			// if _, ok := imports[rpc.Method.Input.GoIdent.GoImportPath]; ok {
			// 	inputStructName = rpc.Method.Input.GoIdent.GoImportPath.Ident(inputStructName)
			// }
			g.Write([]byte(rpc.Method.Comments.Leading.String()))
			g.P(
				"\t",
				rpc.Method.GoName,
				"(",
				contextPackage.Ident("Context"),
				", *",
				rpc.Method.Input.GoIdent,
				") (*",
				rpc.Method.Output.GoIdent,
				", error)",
			)
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
			g.P(
				"func (p *",
				ctrlName,
				")",
				ToPrivateName(rpc.Method.GoName),
				"(ctx *",
				ginPackage.Ident("Context"),
				") {",
			)

			g.P("body := ", rpc.Method.Input.GoIdent, "{}")
			if rpc.HTTPMethod != "GET" {
				// TODO if anything left in body
				g.P("raw, err :=", ioutilPackage.Ident("ReadAll"), "(ctx.Request.Body)")
				g.P("if err != nil {")
				g.P("	ctx.Error(err)")
				g.P("	return")
				g.P("}")
				g.P(protojsonPackage.Ident("Unmarshal"), "(raw, &body)")
			}
			for _, qpm := range rpc.QueryParameters {
				g.P("body.", qpm.ModelParameter, "= ctx.Query(\",", qpm.Key, "\")")
			}
			for _, pth := range rpc.PathParameters {
				g.P("body.", pth.ModelParameter, "= ctx.Param(\",", pth.Key, "\")")
			}

			g.P("var c ", contextPackage.Ident("Context"))
			g.P("if v, ok := ctx.Get(InternalContextKey); ok {")
			g.P("	c, _ = v.(", contextPackage.Ident("Context"), ")")
			g.P("}")
			g.P("if c == nil {")
			g.P("	c = ctx")
			g.P("}")

			g.P("res, err := p.app.", rpc.Method.GoName, "(")
			g.P("c,")
			g.P("&body,")
			g.P(")")
			g.P("if err != nil {")
			g.P("ctx.Error(err)")
			g.P("return")
			g.P("}")

			g.P("resraw, err := protomarsh.Marshal(res)")
			g.P("if err != nil {")
			g.P("	ctx.Error(err)")
			g.P("	return")
			g.P("}")
			g.P("ctx.Status(200)")
			g.P("ctx.Header(\"Content-Type\", \"application/json\")")
			g.P("_, err = ctx.Writer.Write(resraw)")
			g.P("if err != nil {")
			g.P("	ctx.Error(err)")
			g.P("	return")
			g.P("}")
			g.P("}")
		}

		g.P("func Register", srv.Service.GoName, "HTTPServer (")
		g.P("grp *", ginPackage.Ident("RouterGroup"), ",")
		g.P("srv ", intname, ",")
		g.P(") {")
		g.P("ctrl := ", ctrlName, "{app: srv}")
		for _, rpc := range srv.Paths {
			g.P(
				"grp.",
				rpc.HTTPMethod,
				"(\"",
				rpc.Path,
				"\", ",
				"ctrl.",
				ToPrivateName(rpc.Method.GoName),
				")",
			)
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
	g.P("  version: ", "'1.0'") // TODO: better way to figure this out
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
			g.P("      requestBody:")
			g.P("        description: ", api.Method.Input.GoIdent.GoName)
			g.P("        content:")
			g.P("          application/json:")
			g.P("            schema:")
			g.P("              $ref: '#/components/schemas/", api.Method.Input.GoIdent.GoName, "'")
			g.P("        required: true")
			g.P("      responses:")
			g.P("        '200':")
			g.P("          description: ", api.Method.Output.GoIdent.GoName)
			g.P("          content: ")
			g.P("            application/json:")
			g.P("              schema:")
			g.P(
				"                $ref: '#/components/schemas/",
				api.Method.Output.GoIdent.GoName,
				"'",
			)

		}
	}

	g.P("components:")
	g.P("  schemas:")
	schemas := map[string]struct{}{}
	for _, svc := range srvs {
		for _, api := range svc.Paths {

			if err := generateOpenAPIComponentSchema(
				g,
				schemas,
				api.Method.Output,
			); err != nil {
				return err
			}

			if err := generateOpenAPIComponentSchema(
				g,
				schemas,
				api.Method.Input,
			); err != nil {
				return err
			}

		}
	}
	return nil
}

func ToPrivateName(in string) (out string) {
	inr := []rune(in)
	inr[0] = unicode.ToLower(inr[0])
	out = string(inr)
	return
}

func generateOpenAPIComponentSchema(
	g *protogen.GeneratedFile,
	s map[string]struct{},
	m *protogen.Message,
) error {
	foundMessages := []*protogen.Message{}
	if _, ok := s[m.GoIdent.GoName]; !ok {
		s[m.GoIdent.GoName] = struct{}{}
		g.P("    ", m.GoIdent.GoName, ":")
		g.P("      type: object")
		g.P("      properties:")
		for _, fld := range m.Fields {
			field := fld
			g.P("        ", field.Desc.JSONName(), ":")

			prfx := ""
			if field.Desc.IsMap() {
				for _, f := range field.Message.Fields {
					if f.Desc.JSONName() == "value" {
						g.P("          type: object")
						g.P("          additionalProperties:")
						prfx = "  "
						field = f
					}
				}
			}
			if field.Desc.IsList() {
				g.P("          type: array")
				g.P("          items:")
				prfx = "  "
			}

			kind := field.Desc.Kind()
			switch kind {
			case protoreflect.BoolKind:
				g.P(prfx, "          type: boolean")
				g.P(prfx, "          example: false")
			case protoreflect.EnumKind: // TODO
				g.P(prfx, "          type: string")

				values := field.Enum.Values[0].Desc.Name()
				for i := 1; i < len(field.Enum.Values); i++ {
					values = values + ", " + field.Enum.Values[i].Desc.Name()
				}
				g.P(prfx, "          enum: [", values, "]")
			case protoreflect.Int32Kind,
				protoreflect.Sint32Kind,
				protoreflect.Uint32Kind:
				g.P(prfx, "          type: integer")
				g.P(prfx, "          format: int32")
				g.P(prfx, "          example: 1")
			case protoreflect.Int64Kind,
				protoreflect.Sint64Kind,
				protoreflect.Uint64Kind:
				g.P(prfx, "          type: integer")
				g.P(prfx, "          format: int64")
				g.P(prfx, "          example: 1")
			case protoreflect.Sfixed32Kind,
				protoreflect.Fixed32Kind,
				protoreflect.FloatKind:
				g.P(prfx, "          type: number")
				g.P(prfx, "          format: float")
				g.P(prfx, "          example: 1.0")
			case protoreflect.Sfixed64Kind,
				protoreflect.Fixed64Kind,
				protoreflect.DoubleKind:
				g.P(prfx, "          type: number")
				g.P(prfx, "          format: double")
				g.P(prfx, "          example: 1.0")
			case protoreflect.StringKind:
				g.P(prfx, "          type: string")
				g.P(prfx, "          example: sample")
			case protoreflect.BytesKind:
				g.P(prfx, "          type: string")
				g.P(prfx, "          format: byte")
				g.P(prfx, "          example: false")
			case protoreflect.MessageKind:
				if field.Message.Desc.FullName() == "google.protobuf.Timestamp" {
					g.P(prfx, "          type: string")
					g.P(prfx, "          format: date-time")
					g.P(prfx, "          example: '2017-07-21T17:32:28Z'")
				} else if field.Message.Desc.FullName() == "google.protobuf.Struct" {
					g.P(prfx, "          type: object")
				} else {
					foundMessages = append(foundMessages, field.Message)
					g.P(prfx, "          $ref: '#/components/schemas/", field.Message.GoIdent.GoName, "'")
				}

			case protoreflect.GroupKind: // TODO
			}
		}
	}

	for _, found := range foundMessages {
		generateOpenAPIComponentSchema(g, s, found)
	}
	return nil
}
