package telegram

type tokenResponse struct {
	IDToken string `json:"id_token"`
}

type errorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}
