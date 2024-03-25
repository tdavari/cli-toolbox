/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

// dnsCmd represents the dns command
var dnsCmd = &cobra.Command{
	Use:   "dns",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("dns called")
	},
}

func init() {
	netCmd.AddCommand(dnsCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// dnsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// dnsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}





type nslookup struct {
	domain string
	nameservers []*net.NS
}

func (nl nslookup) String() string {
	result := fmt.Sprintf("Domain: %s\nNameservers:\n", nl.domain)
	for _, ns := range nl.nameservers {
		result += fmt.Sprintf("- %s\n", ns.Host)
	}
	return result
}

const workerCount = 900

func main() {
	domains, _ := readFileToList("domain.txt")
    // Create a custom resolver
    r := &net.Resolver{
        PreferGo: true,
        Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
            // Create a Dialer with a timeout of 10 seconds
            d := net.Dialer{
                Timeout: time.Millisecond * time.Duration(1000),
            }
            // Dial the specified DNS server
            return d.DialContext(ctx, network, "8.8.8.8:53")
        },
    }

	// Semaphore to limit the number of concurrent goroutines
	semaphore := make(chan struct{}, workerCount)

	var wg sync.WaitGroup

	// Channel to receive statistics from goroutines
	resultCh := make(chan *nslookup, len(domains))

	// Loop through each domain
	for _, domain := range domains {
		wg.Add(1)

		// Acquire a slot in the semaphore
		semaphore <- struct{}{}
		go func(d string) {
			defer wg.Done()
		

			// Perform DNS lookup for NS (nameserver) records of a domain
			nsRecords, err := r.LookupNS(context.Background(), d)
			if err != nil {
				// fmt.Println("Error:", domain, err)
				<-semaphore
				return
			}

			res := &nslookup{
				domain: d,
				nameservers: nsRecords,
			}
			// fmt.Println(d)
			resultCh <- res

			<-semaphore
		}(domain)
	}

	// Wait for all goroutines to finish
	wg.Wait()
	close(resultCh)

	// Print the nameservers
	for res := range resultCh {
		fmt.Println(res)		
	}
}


func readFileToList(fileName string) ([]string, error) {
	// Open the file
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Initialize an empty list to store lines
	lines := []string{}

	// Create a scanner to read the file line by line
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		// Strip leading and trailing whitespace from each line
		line := scanner.Text()
		line = strings.TrimSpace(line)
		lines = append(lines, line)
	}

	// Check for errors during scanning
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

