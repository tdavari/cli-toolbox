/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/spf13/cobra"
)

// ripeCmd represents the ripe command
var ripeCmd = &cobra.Command{
	Use:   "ripe",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("ripe called")
	},
}

func init() {
	netCmd.AddCommand(ripeCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// ripeCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// ripeCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}




const maxConcurrent = 50 // Control the number of concurrent requests

type BGPState struct {
	TargetPrefix string `json:"target_prefix"`
	Path         []int  `json:"path"`
}

type ResponseData struct {
	BGPState []BGPState `json:"bgp_state"`
}

type Response struct {
	Data ResponseData `json:"data"`
}

type Output struct {
	IP          string `json:"ip"`
	TargetPrefix string `json:"target_prefix"`
	AS          string `json:"as"`
}

func fetchBGPState(ip string, wg *sync.WaitGroup, results chan Output) {
	defer wg.Done()

	url := fmt.Sprintf("https://stat.ripe.net/data/bgp-state/data.json?resource=%s", ip)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Failed to fetch data for IP: %s\n", ip)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var response Response
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			fmt.Printf("Failed to decode response for IP: %s\n", ip)
			return
		}

		if len(response.Data.BGPState) > 0 {
			firstBGPState := response.Data.BGPState[0]
			lastAS := strconv.Itoa(firstBGPState.Path[len(firstBGPState.Path)-1])

			results <- Output{
				IP:          ip,
				TargetPrefix: firstBGPState.TargetPrefix,
				AS:          lastAS,
			}
		} else {
			fmt.Printf("No BGP state information available for IP: %s\n", ip)
		}
	} else {
		fmt.Printf("Failed to retrieve data for IP: %s\n", ip)
	}
}

func ripe() {
	// Read IP addresses from the "ip.txt" file
	file, err := os.Open("ip.txt")
	if err != nil {
		fmt.Printf("Failed to open the file: %v\n", err)
		return
	}
	defer file.Close()

	var ipAddresses []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		ip := scanner.Text()
		ipAddresses = append(ipAddresses, ip)
	}

	results := make(chan Output, len(ipAddresses))
	var wg sync.WaitGroup

	for _, ip := range ipAddresses {
		wg.Add(1)
		go fetchBGPState(ip, &wg, results)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var outputData []Output
	for result := range results {
		outputData = append(outputData, result)
	}

	outputFile := "ip_to_prefix_and_as.json"
	outputJSON, err := json.Marshal(outputData)
	if err != nil {
		fmt.Printf("Failed to marshal output data: %v\n", err)
		return
	}

	file, err = os.Create(outputFile)
	if err != nil {
		fmt.Printf("Failed to create output file: %v\n", err)
		return
	}
	defer file.Close()

	_, err = file.Write(outputJSON)
	if err != nil {
		fmt.Printf("Failed to write output to file: %v\n", err)
	}
}

