/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"
	"github.com/tdavari/cli-toolbox/utils"
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
	Run: ripeMain,
}

var saveToRedis bool // Define a variable to hold the flag value

func init() {
	netCmd.AddCommand(ripeCmd)

	// Here you will define your flags and configuration settings.
	// Define flags for ripe command
	ripeCmd.Flags().StringP("file", "f", "", "File name (required)")
	ripeCmd.MarkFlagRequired("file") // Mark file flag as required

	ripeCmd.Flags().IntP("worker", "w", 3000, "Number of workers")

	// Define optional flag to specify whether to save to Redis
	ripeCmd.Flags().BoolVar(&saveToRedis, "redis", false, "Save results to Redis")

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// ripeCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// ripeCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// Response represents the JSON response structure
type response struct {
	Data struct {
		BGPState []struct {
			Path         []int  `json:"path"`
			TargetPrefix string `json:"target_prefix"`
		} `json:"bgp_state"`
	} `json:"data"`
}

type ripeTask struct {
	IP           string `json:"ip"`
	TargetPrefix string `json:"target_prefix"`
	AS           string `json:"as"`
}

func (r *ripeTask) Process() {
	url := fmt.Sprintf("https://stat.ripe.net/data/bgp-state/data.json?resource=%s", r.IP)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Failed to fetch data for IP: %s\n", r.IP)

		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var response response
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			fmt.Printf("Failed to decode response for IP: %s\n", r.IP)

			return
		}

		if len(response.Data.BGPState) > 0 {
			firstBGPState := response.Data.BGPState[0]
			lastAS := strconv.Itoa(firstBGPState.Path[len(firstBGPState.Path)-1])

			r.TargetPrefix = firstBGPState.TargetPrefix
			r.AS = lastAS

		} else {
			fmt.Printf("No BGP state information available for IP: %s\n", r.IP)
		}
	} else {
		fmt.Printf("Failed to retrieve data for IP: %s\n", r.IP)
	}
}

func ripeMain(cmd *cobra.Command, args []string) {
	// Record start time
	startTime := time.Now()

	// Read IP addresses from the file
	fileName, _ := cmd.Flags().GetString("file")
	workerCount, _ := cmd.Flags().GetInt("worker")

	ips, _ := utils.ReadFileToList(fileName)

	// Create list of tasks
	var tasks []*ripeTask

	for _, ip := range ips {
		// Create RipeTask instance for each IP address
		task := ripeTask{
			IP: ip,
			// You can set other fields as needed
		}

		// Append the task to the tasks slice
		tasks = append(tasks, &task)
	}

	// Create a worker pool
	wp := utils.WorkerPool[*ripeTask]{
		Tasks:       tasks,
		Concurrency: workerCount, // Number of workers that can run at a time
	}

	// Run the pool
	wp.Run()
	fmt.Println("All tasks have been processed!")

	// Save tasks result
	if saveToRedis {
		// Save results to Redis hash map
		saveToRedisHash(tasks, "ripe_task")
	} else {
		// Save results to a file
		saveToFile(tasks, fileName)
	}

	// Calculate and print elapsed time
	fmt.Printf("Execution time: %s\n", time.Since(startTime))
}

func saveToFile(tasks []*ripeTask, fileName string) {
	// Save tasks result to a file
	outputFile := getOutputFileName(fileName)
	outputJSON, err := json.Marshal(tasks)
	if err != nil {
		fmt.Printf("Failed to marshal output data: %v\n", err)
		return
	}

	file, err := os.Create(outputFile)
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

func saveToRedisHash(tasks []*ripeTask, hmapName string) error {
	// Initialize Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr:     "1.1.1.1:6379",
		Password: "", // no password set
		DB:       11, // use default DB
	})
	defer rdb.Close()

	// Prepare a map to hold task IPs and their corresponding JSON representations
	taskMap := make(map[string]interface{})
	for _, task := range tasks {
		taskJSON, err := json.Marshal(task)
		if err != nil {
			return err
		}
		taskMap[task.IP] = taskJSON
	}

	// Store all tasks in Redis hash map
	ctx := context.Background()
	err := rdb.HMSet(ctx, hmapName, taskMap).Err()
	if err != nil {
		return err
	}

	return nil
}

func getOutputFileName(name string) string {
	parts := strings.Split(name, ".")
	if len(parts) > 1 {
		name = parts[0]
	}
	return name + ".json"
}
