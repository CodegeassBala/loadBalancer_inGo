package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type Config struct {
    Port    string
    Servers []string
	Max_connections int;
	Timeout time.Duration;
}

type Server struct{
	// URL is public
	URL string

	healthy bool
}

type LoadBalancer struct{
	servers []*Server

	// current_idx is private.
	current_idx int

	ConnectionPool ConnectionPool

	mu sync.Mutex
}

func NewLoadBalancer(servers []string,opts *Opts) *LoadBalancer{
	Servers := make([]*Server,len(servers));
	fmt.Println(len(Servers));
	for idx,server:=range servers{
		Servers[idx] = &Server{
			URL:server,
			healthy: true,
		}
	}
	return &LoadBalancer{
		servers: Servers,
		current_idx:0,
		ConnectionPool: *NewConnectionPool(opts),
	}
}

func (lb *LoadBalancer)  ServeHTTP(writer http.ResponseWriter, req *http.Request){
	 next_server,err := lb.NextServer()
	 if(err!=nil){
		panic(err)
	 }
	 res,err := lb.ForwardRequest(next_server,req.RequestURI)
	 if err!=nil{
		panic(err)
	 }
	 defer res.Body.Close()
	 
	 body,err := io.ReadAll(res.Body)
	 if err!=nil{
		panic(err)
	 }
	 _,err = writer.Write(body)
	 if err!=nil{
		panic(err)
	 }
}

func (lb *LoadBalancer) NextServer() (*Server,error){
	lb.mu.Lock()
	defer lb.mu.Unlock()
	idx:=lb.current_idx;
	count:=0;
	num_servers := len(lb.servers)
	for count< num_servers {
		if lb.servers[idx].healthy{
			break;
		}
		idx = (idx+1)%(num_servers)
		count++;
	}
	if count == num_servers{
		return nil,errors.New("no healthy servers");
	}
	next_server := lb.servers[idx];
	lb.current_idx = (idx+1)%(num_servers)
	return next_server,nil
}

func (lb *LoadBalancer) ForwardRequest(server *Server, uri string) (*http.Response,error){
	base_url,err := url.Parse(server.URL)
	if err!=nil {
		return nil,err
	}
	fmt.Println("Forwarding request to",server.URL)
	full_url :=  base_url.ResolveReference(&url.URL{Path: uri})
	client := lb.ConnectionPool.Get(server.URL);
	http_response,err := client.Get(full_url.String())
	if err!=nil{
		return nil,err
	}
	lb.ConnectionPool.Put(server.URL,client);
	return http_response,nil
}
func (lb *LoadBalancer) HealthCheck() {
	for _,server := range lb.servers{
		response,err := http.Get(server.URL+ "/health_check");
		if err!=nil || response.StatusCode != http.StatusOK {
			server.healthy = false;
			log.Println("!!",server.URL,"is DOWN..!");
		} else {
			server.healthy = true;
			log.Println("!!",server.URL,"is UP..!");
		}	
	}
}

func (lb *LoadBalancer) RunHealthCheck(){
	 ticker := time.NewTicker(5*time.Second);

	 go func(){
		for range ticker.C{
			lb.HealthCheck()
		}
	 }();
}
type Opts struct{
	max_connections int;
	time_out time.Duration;
}
func NewOpts(max_connections int,
	time_out time.Duration) *Opts{
		return &Opts{
			max_connections,
			time_out,
		}
}
type ConnectionPool struct{
	*Opts
	clients map[string][]*http.Client
	mu sync.Mutex
}

func NewConnectionPool(opts *Opts) *ConnectionPool{
	return &ConnectionPool{
		Opts: opts,
		clients: make(map[string][]*http.Client),
	}
}

func (cp *ConnectionPool) Get(server string) *http.Client{
	cp.mu.Lock()
	defer cp.mu.Unlock()
	clients,ok:=cp.clients[server];
	if  ok && len(clients)>0{
		client := clients[len(clients)-1];
		clients  = clients[:len(clients)-1];
		cp.clients[server] = clients;
		return client; 
	}
	
	return &http.Client{
		Timeout: cp.time_out,
	}
}

func (cp *ConnectionPool) Put(server string,client *http.Client) error{
	cp.mu.Lock()
	defer cp.mu.Unlock()
	clients,ok:=cp.clients[server];
	if(ok && cp.max_connections > len(clients)){
		clients = append(clients, client);
		cp.clients[server] = clients;
		return nil;
	}
	return errors.New("invalid put operation either the server is invalid or the connection pool is full")
}

func main(){
	fmt.Println("Load Balancer !!!!")
	config,err := ParseConfig();
	if err!=nil{
		panic(err);
	}
	fmt.Println(config);
	lb := NewLoadBalancer(config.Servers,NewOpts(config.Max_connections,config.Timeout));
	
	lb.RunHealthCheck()
	err = http.ListenAndServe(":"+config.Port,lb);
	if err!=nil{
		panic(err)
	}
	
}