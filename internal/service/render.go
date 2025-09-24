package service

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
)

type Renderer interface {
	RenderHome(http.ResponseWriter)
}

type Render struct {
	homeTemplate *template.Template
	logger       *slog.Logger
}

func New(templatePath string, logger *slog.Logger) *Render {
	return &Render{
		homeTemplate: template.Must(template.ParseFiles(fmt.Sprintf("%s/%s", templatePath, "home.html"))),
		logger:       logger,
	}
}

func (r *Render) RenderHome(w http.ResponseWriter) {
	err := r.homeTemplate.Execute(w, nil)
	if err != nil {
		r.logger.Error("can not execute home page", slog.String("error", err.Error()))
	}
}
