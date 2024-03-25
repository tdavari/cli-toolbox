/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"sync"
	"time"

	probing "github.com/prometheus-community/pro-bing"

	"github.com/spf13/cobra"
	"github.com/tdavari/cli-toolbox/utils"
)

// pingCmd represents the ping command
var pingCmd = &cobra.Command{
	Use:   "ping",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: ping,
}

func init() {
	netCmd.AddCommand(pingCmd)

	// Define flags for ping command
	pingCmd.Flags().StringP("file", "f", "", "File name (required)")
	pingCmd.MarkFlagRequired("file") // Mark file flag as required

	pingCmd.Flags().IntP("worker", "w", 900, "Number of workers")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// pingCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// pingCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

const WorkersCount = 1000
const fileName = ""

// removeDuplicate in a generic (int, sting , ...) list
func removeDuplicate[T comparable](sliceList []T) []T {
	allKeys := make(map[T]bool)
	list := []T{}
	for _, item := range sliceList {

		_, exists := allKeys[item]

		if !exists {
			allKeys[item] = true
			list = append(list, item)
		}
	}

	return list
}

// readFileToList reads a file and returns a list of strings where each string is a line from the file
// func readFileToList(fileName string) ([]string, error) {
// 	// Open the file
// 	file, err := os.Open(fileName)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer file.Close()

// 	// Initialize an empty list to store lines
// 	lines := []string{}

// 	// Create a scanner to read the file line by line
// 	scanner := bufio.NewScanner(file)
// 	for scanner.Scan() {
// 		// Strip leading and trailing whitespace from each line
// 		line := scanner.Text()
// 		line = strings.TrimSpace(line)
// 		lines = append(lines, line)
// 	}

// 	// Check for errors during scanning
// 	if err := scanner.Err(); err != nil {
// 		return nil, err
// 	}

// 	return lines, nil
// }

func ping(cmd *cobra.Command, args []string) {
	defer timer("main")()

	fileName, _ := cmd.Flags().GetString("file")
	workerCount, _ := cmd.Flags().GetInt("worker")

	ipAddresses, err := utils.ReadFileToList(fileName)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	ipAddresses = removeDuplicate(ipAddresses)

	// Semaphore to limit the number of concurrent goroutines
	semaphore := make(chan struct{}, workerCount)

	var wg sync.WaitGroup

	// Channel to receive statistics from goroutines
	statsCh := make(chan *probing.Statistics, len(ipAddresses))

	// Loop through each IP address and spawn a goroutine
	for _, ip := range ipAddresses {
		wg.Add(1)

		// Acquire a slot in the semaphore
		semaphore <- struct{}{}
		go func(ip string) {
			defer wg.Done()

			pinger, err := probing.NewPinger(ip)
			if err != nil {
				fmt.Printf("Error creating pinger for IP %s: %v\n", ip, err)
				return
			}
			// pinger.SetPrivileged(true)
			pinger.Count = 3
			pinger.Timeout = 3 * time.Second
			err = pinger.Run() // Blocks until finished.
			if err != nil {
				fmt.Printf("Error running pinger for IP %s: %v\n", ip, err)
				return
			}
			stats := pinger.Statistics() // get send/receive/duplicate/rtt stats
			statsCh <- stats

			<-semaphore
		}(ip)
	}

	// Wait for all goroutines to finish

	wg.Wait()
	close(statsCh)

	successCount := 0
	failureCount := 0
	// Read statistics from the channel and print
	for stats := range statsCh {
		// fmt.Printf("%+v\n", stats)
		if len(stats.Rtts) > 0 {
			successCount++
		} else {
			failureCount++
		}
	}
	fmt.Printf("Number of IPs that pinged: %d\n", successCount)
	fmt.Printf("Number of IPs that did not ping: %d\n", failureCount)

}

func timer(name string) func() {
	start := time.Now()
	return func() {
		fmt.Printf("%s took %v\n", name, time.Since(start))
	}
}
