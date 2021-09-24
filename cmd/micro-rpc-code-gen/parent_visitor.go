package main

import "github.com/emicklei/proto"

type parentVisitor struct {
	cb func(v *Parent)
}

const ( // parent type
	EPT_Service = iota
	EPT_RPC
	EPT_Message
)

type Parent struct {
	Name    string
	Comment *proto.Comment
	Type    int // see EPT_* constants
}

func (pr *parentVisitor) VisitMessage(m *proto.Message) {
	pr.cb(&Parent{
		Name:    m.Name,
		Comment: m.Comment,
		Type:    EPT_Message,
	})
}

func (pr *parentVisitor) VisitService(v *proto.Service) {
	pr.cb(&Parent{
		Name:    v.Name,
		Comment: v.Comment,
		Type:    EPT_Service,
	})
}

func (pr *parentVisitor) VisitSyntax(s *proto.Syntax) {
	// nop
}

func (pr *parentVisitor) VisitPackage(p *proto.Package) {
	//pr.cb(&Parent{
	//	Name:    p.Name,
	//	Comment: p.Comment,
	//})
}

func (pr *parentVisitor) VisitOption(o *proto.Option) {
	//pr.cb(&Parent{
	//	Name:    o.Name,
	//	Comment: o.Comment,
	//})
}

func (pr *parentVisitor) VisitImport(i *proto.Import) {
	// nop
}

func (pr *parentVisitor) VisitNormalField(i *proto.NormalField) {
	// nop
}

func (pr *parentVisitor) VisitEnumField(i *proto.EnumField) {
	// nop
}

func (pr *parentVisitor) VisitEnum(e *proto.Enum) {
	// nop
}

func (pr *parentVisitor) VisitComment(e *proto.Comment) {
	// nop
}

func (pr *parentVisitor) VisitOneof(o *proto.Oneof) {
	// nop
}

func (pr *parentVisitor) VisitOneofField(o *proto.OneOfField) {
	// nop
}

func (pr *parentVisitor) VisitReserved(r *proto.Reserved) {
	// nop
}

func (pr *parentVisitor) VisitRPC(r *proto.RPC) {
	pr.cb(&Parent{
		Name:    r.Name,
		Comment: r.Comment,
		Type:    EPT_RPC,
	})
}

func (pr *parentVisitor) VisitMapField(f *proto.MapField) {
	// nop
}

func (pr *parentVisitor) VisitGroup(g *proto.Group) {
	// nop
}

func (pr *parentVisitor) VisitExtensions(e *proto.Extensions) {
	// nop
}
