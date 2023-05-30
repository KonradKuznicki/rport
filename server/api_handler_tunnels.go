package chserver

import (
	"net/http"
	"time"

	"github.com/realvnc-labs/rport/server/api"
	"github.com/realvnc-labs/rport/server/clients/clienttunnel"
	"github.com/realvnc-labs/rport/share/models"
)

type TunnelPayload struct {
	models.Remote
	ID        string    `json:"id"`
	ClientID  string    `json:"client_id"`
	CreatedAt time.Time `json:"created_at"`
}

func convertToTunnelPayload(t *clienttunnel.Tunnel, clientID string) TunnelPayload {
	return TunnelPayload{
		Remote:    t.Remote,
		ID:        t.ID,
		ClientID:  clientID,
		CreatedAt: t.CreatedAt,
	}
}

func (al *APIListener) handleGetTunnels(w http.ResponseWriter, req *http.Request) {
	curUser, err := al.getUserModelForAuth(req.Context())
	if err != nil {
		al.jsonError(w, err)
		return
	}

	clientGroups, err := al.clientGroupProvider.GetAll(req.Context())
	if err != nil {
		al.jsonError(w, err)
	}

	clients, err := al.clientService.GetUserClients(clientGroups, curUser)
	if err != nil {
		al.jsonError(w, err)
	}

	tunnels := make([]TunnelPayload, 0)
	for _, c := range clients {
		clientID := c.GetID()
		if !c.IsConnected() {
			continue
		}

		for _, t := range c.GetTunnels() {
			tunnels = append(tunnels, convertToTunnelPayload(t, clientID))
		}
	}

	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(tunnels))
}
