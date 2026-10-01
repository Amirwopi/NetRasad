package diagnostics

type PingResult struct {
	Host       string
	Sent       int
	Received   int
	PacketLoss float64
	MinRTT     float64
	AvgRTT     float64
	MaxRTT     float64
	RTTs       []float64
	Output     string
}

type TracerouteHop struct {
	Hop  int
	Host string
	IP   string
	RTT1 float64
	RTT2 float64
	RTT3 float64
}

type TracerouteResult struct {
	Hops   []TracerouteHop
	Output string
}

type DNSResult struct {
	Host        string
	ARecords    []string
	AAAARecords []string
	NSRecords   []string
	MXRecords   []string
	CNAME       string
}

type TCPResult struct {
	Host    string
	Port    int
	Success bool
	RTT     float64
	Error   string
}

type GatewayInfo struct {
	GatewayIP string
	Interface string
	Metric    int
}

type Service struct{}

func NewService() *Service { return &Service{} }
