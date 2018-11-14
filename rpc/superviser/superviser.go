// Generated by go-rpcgen. Do not modify.

package superviser

import (
	. "github.com/SamsungSLAV/boruta"
	"net/rpc"
)

// SuperviserService is generated service for Superviser interface.
type SuperviserService struct {
	impl Superviser
}

// NewSuperviserService creates a new SuperviserService instance.
func NewSuperviserService(impl Superviser) *SuperviserService {
	return &SuperviserService{impl}
}

// RegisterSuperviserService registers impl in server.
func RegisterSuperviserService(server *rpc.Server, impl Superviser) error {
	return server.RegisterName("Superviser", NewSuperviserService(impl))
}

// SuperviserRegisterRequest is a helper structure for Register method.
type SuperviserRegisterRequest struct {
	Caps         Capabilities
	DryadAddress string
	SshAddress   string
}

// SuperviserRegisterResponse is a helper structure for Register method.
type SuperviserRegisterResponse struct {
}

// Register is RPC implementation of Register calling it.
func (s *SuperviserService) Register(request *SuperviserRegisterRequest, response *SuperviserRegisterResponse) (err error) {
	err = s.impl.Register(request.Caps, request.DryadAddress, request.SshAddress)
	return
}

// SuperviserSetFailRequest is a helper structure for SetFail method.
type SuperviserSetFailRequest struct {
	Uuid   WorkerUUID
	Reason string
}

// SuperviserSetFailResponse is a helper structure for SetFail method.
type SuperviserSetFailResponse struct {
}

// SetFail is RPC implementation of SetFail calling it.
func (s *SuperviserService) SetFail(request *SuperviserSetFailRequest, response *SuperviserSetFailResponse) (err error) {
	err = s.impl.SetFail(request.Uuid, request.Reason)
	return
}

// SuperviserClient is generated client for Superviser interface.
type SuperviserClient struct {
	client *rpc.Client
}

// DialSuperviserClient connects to addr and creates a new SuperviserClient instance.
func DialSuperviserClient(addr string) (*SuperviserClient, error) {
	client, err := rpc.Dial("tcp", addr)
	return &SuperviserClient{client}, err
}

// NewSuperviserClient creates a new SuperviserClient instance.
func NewSuperviserClient(client *rpc.Client) *SuperviserClient {
	return &SuperviserClient{client}
}

// Close terminates the connection.
func (_c *SuperviserClient) Close() error {
	return _c.client.Close()
}

// Register is part of implementation of Superviser calling corresponding method on RPC server.
func (_c *SuperviserClient) Register(caps Capabilities, dryadAddress string, sshAddress string) (err error) {
	_request := &SuperviserRegisterRequest{caps, dryadAddress, sshAddress}
	_response := &SuperviserRegisterResponse{}
	err = _c.client.Call("Superviser.Register", _request, _response)
	return err
}

// SetFail is part of implementation of Superviser calling corresponding method on RPC server.
func (_c *SuperviserClient) SetFail(uuid WorkerUUID, reason string) (err error) {
	_request := &SuperviserSetFailRequest{uuid, reason}
	_response := &SuperviserSetFailResponse{}
	err = _c.client.Call("Superviser.SetFail", _request, _response)
	return err
}