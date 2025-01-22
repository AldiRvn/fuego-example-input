package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rs/cors"

	"github.com/swaggest/swgui/v5emb"

	"github.com/go-fuego/fuego"

	_ "github.com/joho/godotenv/autoload" //? load file .env
	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
)

func init() {
	//? Config Logger
	w := os.Stdout
	slog.SetDefault(slog.New(
		tint.NewHandler(w, &tint.Options{
			NoColor: !isatty.IsTerminal(w.Fd()),
		}),
	))
}

type Request struct {
	Name string `json:"name"`
}

type Response struct {
	Message       string `json:"message"`
	Datum         any    `json:"datum,omitempty"`
	Data          []any  `json:"data,omitempty"`
	TimeExecution string `json:"timeExecution"`
}

func DeferController(start time.Time, res *Response) {
	res.TimeExecution = time.Since(start).String()
}

func main() {
	service := fuego.NewServer(
		fuego.WithAddr("0.0.0.0:"+os.Getenv("APP_PORT")), //? Setting port
		fuego.WithCorsMiddleware(cors.New(cors.Options{ //? Setting CORS
			AllowedOrigins: strings.Split(os.Getenv("CORS_ALLOWED_ORIGIN"), ","),
			AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
		}).Handler),
	)
	desc := service.Engine.OpenAPI.Description()
	desc.Info.Description = "" //? Override default description

	service = fuego.Group(service, "/", fuego.OptionHeader("Authorization", "Bearer")) //? Apply header to all endpoint
	//* --------------------------- MIDDLEWARE EXAMPLE --------------------------- */
	fuego.Use(service, func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			slog.Info("request", "uri", r.RequestURI)
			next.ServeHTTP(w, r)
		})
	})

	pathSwagger := "/" //? Setting swagger ui
	fuego.Handle(service, pathSwagger, v5emb.New("Swagger UI V5", "/swagger/openapi.json", pathSwagger))
	slog.Info("Swagger UI V5", "web", "http://"+service.Addr+pathSwagger)

	//* --------------------------- EXAMPLE QUERY+PATH PARAM -------------------------- */
	fuego.Get(service, "/get/{page}", func(req fuego.ContextNoBody) (res Response, err error) {
		defer DeferController(time.Now(), &res)
		res.Message = "OK"

		page := req.PathParam("page")
		search := req.QueryParam("search")
		size := req.QueryParamInt("size")
		res.Datum = map[string]any{
			"param": map[string]any{
				"page":   page,
				"search": search,
				"size":   size,
			},
		}
		slog.Info("query param",
			"page", page,
			"search", search,
			"size", size,
		)
		return
	},
		fuego.OptionPath("page", "Example param path", fuego.ParamDefault("1")),
		fuego.OptionQuery("search", "Example param query", fuego.ParamDefault("Time")),
		fuego.OptionQueryInt("size", "Example param query int", fuego.ParamDefault(10)),
	)

	//? group multiple enpoint into 1 tag
	post := fuego.Group(service, "/post")
	//* --------------------- EXAMPLE POST WITHOUT BODY PARAM -------------------- */
	fuego.Post(post, "/withoutBody", func(req fuego.ContextNoBody) (res Response, err error) {
		defer DeferController(time.Now(), &res)
		res.Message = "OK"
		return
	})

	//* --------------------- EXAMPLE POST WITH BODY PARAM -------------------- */
	fuego.Post(post, "/withBody", func(req fuego.ContextWithBody[Request]) (res Response, err error) {
		defer DeferController(time.Now(), &res)
		realReq, err := req.Body()
		if err != nil {
			log.Println(err)
			return
		}
		res.Message = "Hi " + realReq.Name
		return
	})

	service.Run()
}
