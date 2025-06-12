package api

import (
	"context"
	"net/http"
	"time"


	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/gunjanjp/gunj-operator/internal/api/graphql/generated"
	"github.com/gunjanjp/gunj-operator/internal/api/graphql/resolvers"
)

// setupGraphQL configures GraphQL endpoints
func (s *Server) setupGraphQL() {
	// Create GraphQL server
	resolver := resolvers.NewResolver(s.client, s.log)
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: resolver}))

	// Configure GraphQL server
	srv.AddTransport(&transport.Websocket{
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Configure origin checking for production
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		KeepAlivePingInterval: 10 * time.Second,
	})
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{})

	// Add extensions
	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New(100),
	})

	// GraphQL endpoint
	s.router.POST("/graphql", func(c *gin.Context) {
		// Pass authentication context to GraphQL
		ctx := c.Request.Context()
		if user, exists := c.Get("user"); exists {
			ctx = context.WithValue(ctx, "user", user)
		}
		if groups, exists := c.Get("groups"); exists {
			ctx = context.WithValue(ctx, "groups", groups)
		}

		srv.ServeHTTP(c.Writer, c.Request.WithContext(ctx))
	})

	// GraphQL playground (only in development)
	if s.config.EnableGraphQLPlayground {
		s.router.GET("/playground", func(c *gin.Context) {
			playground.Handler("GraphQL", "/graphql").ServeHTTP(c.Writer, c.Request)
		})
	}
}

// GraphQL Schema example (would be in schema.graphqls file)
const schemaExample = `
type Query {
    platforms(namespace: String, limit: Int, offset: Int): PlatformConnection!
    platform(name: String!, namespace: String!): Platform
    platformMetrics(name: String!, namespace: String!, range: TimeRange!): PlatformMetrics!
    currentUser: User!
}

type Mutation {
    createPlatform(input: CreatePlatformInput!): Platform!
    updatePlatform(name: String!, namespace: String!, input: UpdatePlatformInput!): Platform!
    deletePlatform(name: String!, namespace: String!): Boolean!
    backupPlatform(name: String!, namespace: String!, destination: String!): BackupJob!
}

type Subscription {
    platformStatus(name: String!, namespace: String!): PlatformStatus!
    platformEvents(name: String, namespace: String): Event!
    metrics(name: String!, namespace: String!, component: ComponentType!): MetricUpdate!
}

type Platform {
    metadata: ObjectMeta!
    spec: PlatformSpec!
    status: PlatformStatus!
    components: [Component!]!
    metrics(range: TimeRange!): PlatformMetrics!
    events(limit: Int): [Event!]!
}
`