package gosocket

type ArgsRequest struct {
	Id   string      `json:"id"`
	Args interface{} `json:"args"`
}

type ArgsResponse struct {
	Id   string   `json:"id"`
	Args Response `json:"args"`
}

type Response struct {
	Result  bool        `json:"result"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type response struct {
	identity string
	client   ClientFace
	event    string
	Response
}

func NewResponse(client ClientFace, event, identity string) (r *response) {
	r = new(response)
	r.client = client
	r.event = event
	r.identity = identity
	return
}

func (r *response) Set(message string, result bool, data interface{}) {
	r.Result = result
	r.Message = message
	r.Data = data
	return
}

func (r *response) Success(data interface{}) {
	r.Set("ok", true, data)
	r.emit()
}

func (r *response) Fail(message string) {
	r.Set(message, false, nil)
	r.emit()
}

func (r *response) emit() {
	var args ArgsResponse
	args.Id = r.identity
	args.Args = r.Response
	r.client.Emit(r.event, args)
}
