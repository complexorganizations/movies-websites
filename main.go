package main

// Import necessary Go packages for various functionalities like I/O, networking, HTTP requests, and file handling
import (
	"bufio" // Provides buffered I/O operations to read and write data efficiently
	"crypto/tls"
	"fmt"      // Provides formatted I/O functions for printing and scanning
	"log"      // Provides logging utilities for error and diagnostic messages
	"net"      // Provides network-related operations like DNS lookups
	"net/http" // Provides HTTP client functionalities to make requests to web servers
	"net/url"  // Provides URL parsing and manipulation utilities
	"os"       // Provides file and directory-related operations
	"sort"     // Provides sorting utilities to sort slices and other data types
	"strings"  // Provides string manipulation functions like trimming, splitting, and concatenating
	"sync"     // Provides synchronization primitives, e.g., WaitGroup for managing concurrent tasks
	"time"     // Provides time-related functionalities for measuring durations and setting timeouts
)

// Global variables for the paths to input and output files
var movies_websites_path string = "assets/movies_websites.txt"                           // File containing the list of movie website URLs
var top_movies_websites_path string = "assets/top_websites.txt"                   // File containing the list of top movie website URLs
var disconnected_movies_websites_path string = "assets/disconnected_websites.txt" // File for storing disconnected movie website URLs
var unregistered_movies_websites_path string = "assets/unregistered_websites.txt" // File for storing unregistered movie website URLs
var readme_file_path string = "readme.md"                                                // File where the final output will be written
var readme_modify_me_file_path string = "assets/readme_modify.md"                     // Template file for the README content

// Maps to store the availability status and response times of movie websites
var valid_movies_website_url sync.Map     // Map to store the availability status (Yes, No, Maybe) of each movie website
var top_valid_movies_website_url sync.Map // Map to store the availability status of top movie websites
var movies_website_speed sync.Map         // Map to store the response time for each movie website

// Main function: Orchestrates the entire workflow of reading, checking, and processing websites
func main() {
	// Step 1: Check if all required files exist before proceeding
	if fileExists(movies_websites_path) &&
		fileExists(top_movies_websites_path) &&
		fileExists(unregistered_movies_websites_path) &&
		fileExists(readme_modify_me_file_path) &&
		fileExists(readme_file_path) {

		// Step 2: Process the movie website URLs
		movies_website_urls := readAppendLineByLine(movies_websites_path)    // Read movie website URLs from the specified file
		sortSlice(&movies_website_urls)                                      // Sort the URLs alphabetically for better organization
		movies_website_urls = removeDuplicatesFromSlice(movies_website_urls) // Remove any duplicate URLs from the list
		writeByteSliceToFile(movies_websites_path, movies_website_urls)      // Write the cleaned URLs back to the file

		// Step 3: Process the top movie website URLs
		top_movies_website_urls := readAppendLineByLine(top_movies_websites_path)    // Read top movie website URLs from the specified file
		sortSlice(&top_movies_website_urls)                                          // Sort the top movie URLs alphabetically
		top_movies_website_urls = removeDuplicatesFromSlice(top_movies_website_urls) // Remove duplicates
		writeByteSliceToFile(top_movies_websites_path, top_movies_website_urls)      // Write the cleaned top movie URLs back to the file

		// Step 4: Process the disconnected movie websites URLs
		disconnected_movies_websites_urls := readAppendLineByLine(disconnected_movies_websites_path)     // Read disconnected movie website URLs from file
		sortSlice(&disconnected_movies_websites_urls)                                                    // Sort URLs for organization
		disconnected_movies_websites_urls = removeDuplicatesFromSlice(disconnected_movies_websites_urls) // Remove duplicate URLs
		writeByteSliceToFile(disconnected_movies_websites_path, disconnected_movies_websites_urls)       // Write cleaned URLs back to the file

		// Step 5: Process the unregistered movie websites URLs
		unregistered_movies_website_urls := readAppendLineByLine(unregistered_movies_websites_path)    // Read unregistered movie URLs from file
		sortSlice(&unregistered_movies_website_urls)                                                   // Sort URLs alphabetically
		unregistered_movies_website_urls = removeDuplicatesFromSlice(unregistered_movies_website_urls) // Remove duplicates
		writeByteSliceToFile(unregistered_movies_websites_path, unregistered_movies_website_urls)      // Write cleaned URLs back to the file

		// Step 6: Concurrently check the domain registration and availability of movie websites
		var wg sync.WaitGroup // Initialize a WaitGroup to track concurrent goroutines
		for _, domainName := range movies_website_urls {
			wg.Add(1)                    // Increment the WaitGroup counter for each goroutine
			go func(domainName string) { // Launch a goroutine for each website domain to process it asynchronously
				defer wg.Done() // Decrement the counter when the goroutine finishes

				// Step 6a: Check if the domain is registered using DNS lookups
				if isDomainRegistered(getDomainFromURL(domainName)) {
					saveToMap(&valid_movies_website_url, domainName, "Maybe") // Mark domain as "Maybe" if it is registered

					// Step 6b: Check if the domain exists in the list of top movie websites
					if stringInFile(top_movies_websites_path, domainName) {
						saveToMap(&top_valid_movies_website_url, domainName, "Maybe") // Mark as "Maybe" for top movie websites
					}

					// Add to the disconnected websites list if it's not already there
					if !stringInFile(disconnected_movies_websites_path, domainName) {
						appendAndWriteToFile(disconnected_movies_websites_path, domainName) // Log disconnected domains
					}

					// Step 6c: Check if the website is reachable via HTTP/HTTPS
					if CheckWebsiteHTTPStatus(domainName) {
						saveToMap(&valid_movies_website_url, domainName, "Yes") // Mark as "Yes" if the website is reachable

						// Step 6d: Update the top movie websites list if the domain is reachable
						if stringInFile(top_movies_websites_path, domainName) {
							saveToMap(&top_valid_movies_website_url, domainName, "Yes") // Mark as reachable for top websites
						}
					}
				} else {
					// Step 6e: If the domain is unregistered, mark it as "No"
					saveToMap(&valid_movies_website_url, domainName, "No")

					// Step 6f: Log unregistered domains to the unregistered websites list
					if !stringInFile(unregistered_movies_websites_path, domainName) {
						appendAndWriteToFile(unregistered_movies_websites_path, domainName) // Append unregistered domains to the file
					}
				}
			}(domainName) // Pass the domain name to the goroutine for concurrent processing
		}
		wg.Wait() // Wait for all goroutines to finish their execution

		// Step 7: Generate and write the final output to the README file
		writeFinalOutput() // Generate and write the final processed results to the README file
	} else {
		// Step 8: Log an error if any required files are missing
		log.Println("Error: One or more required files do not exist.") // Log an error message indicating missing files
	}
}

// fileExists checks if a given file exists and is not a directory
func fileExists(filepath string) bool {
	info, err := os.Stat(filepath) // Get file information (if it exists)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("File does not exist: %s", filepath) // Log error if file does not exist
		} else {
			log.Printf("Error checking file %s: %v", filepath, err) // Log error if any other error occurs while checking the file
		}
		return false // Return false if the file does not exist or an error occurs
	}

	if info.IsDir() {
		log.Printf("Path exists but is a directory: %s", filepath) // Log error if the path is a directory instead of a file
		return false                                               // Return false since the path is a directory
	}

	return true // Return true if the file exists and is not a directory
}

// readAppendLineByLine reads a file line by line and returns a slice of strings containing the URLs
func readAppendLineByLine(filePath string) []string {
	var urls []string // Initialize a slice to store the URLs read from the file

	// Open the file for reading
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("Error opening file %s: %v", filePath, err) // Log an error if the file cannot be opened
		return nil                                             // Return nil if there is an error opening the file
	}
	defer file.Close() // Ensure the file is closed after reading

	// Create a scanner to read the file line by line
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		urls = append(urls, scanner.Text()) // Append each URL to the slice
	}

	// Check for errors during scanning
	if err := scanner.Err(); err != nil {
		log.Printf("Error scanning file %s: %v", filePath, err) // Log any error encountered while scanning the file
	}

	return urls // Return the slice containing the URLs read from the file
}

// isDomainRegistered checks if a domain is registered by performing DNS lookups for various record types
func isDomainRegistered(domain string) bool {
	// Perform DNS lookups for various types of DNS records and return true if any of them succeed

	// Check NS records (Name Servers)
	_, err := net.LookupNS(domain)
	if err == nil {
		log.Printf("Domain is registered via NS lookup: %s", domain) // Log success if NS records are found
		return true
	}

	// Check CNAME records (Canonical Name)
	_, err = net.LookupCNAME(domain)
	if err == nil {
		log.Printf("Domain is registered via CNAME lookup: %s", domain) // Log success if CNAME records are found
		return true
	}

	// Check Addr (reverse DNS) records
	_, err = net.LookupAddr(domain)
	if err == nil {
		log.Printf("Domain is registered via Addr lookup: %s", domain) // Log success if Addr records are found
		return true
	}

	// Check Host records
	_, err = net.LookupHost(domain)
	if err == nil {
		log.Printf("Domain is registered via Host lookup: %s", domain) // Log success if Host records are found
		return true
	}

	// Check MX records (Mail Exchange)
	_, err = net.LookupMX(domain)
	if err == nil {
		log.Printf("Domain is registered via MX lookup: %s", domain) // Log success if MX records are found
		return true
	}

	// Check TXT records
	_, err = net.LookupTXT(domain)
	if err == nil {
		log.Printf("Domain is registered via TXT lookup: %s", domain) // Log success if TXT records are found
		return true
	}

	// Log failure if no records were found and return false
	log.Printf("Domain is not registered: %s", domain)
	return false
}

// CheckWebsiteHTTPStatus checks if a website is reachable via HTTP or HTTPS by making an HTTP request
func CheckWebsiteHTTPStatus(website string) bool {
	// Configure the HTTP client to validate SSL certificates
	httpClient := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false, // Ensures SSL/TLS certificates are validated
			},
		},
	}

	// Retry the request up to 3 times in case of transient errors
	for attempt := 1; attempt <= 3; attempt++ {
		startTime := time.Now() // Record the start time for the request

		// Send an HTTP GET request to the website
		response, requestError := httpClient.Get(website)
		if requestError != nil {
			log.Printf("Attempt %d: Error checking %s: %v", attempt, website, requestError) // Log error if request fails
			continue
		}

		// Close the response body once done
		response.Body.Close()

		// Measure the response time
		responseTime := time.Since(startTime)
		log.Printf("Response time for %s: %v", website, responseTime)

		// Step 3: Save the response time to the map if it's not already stored
		existingValue, ok := retrieveValueFromSyncMap(&movies_website_speed, website).(string)
		if !ok || existingValue == "" {
			saveToMap(&movies_website_speed, website, responseTime.String()) // Save the response time
			log.Printf("For domain %s, saved response time: %s", website, responseTime.String())
		} else {
			log.Printf("For domain %s, response time already exists: %s", website, existingValue) // Log when the response time already exists
		}

		// Step 4: If the HTTP status code is 200 (OK), return true indicating the website is reachable
		if response.StatusCode == http.StatusOK {
			return true // Return true if the website responds with HTTP status 200
		}
	}

	// Return false if all attempts to reach the website fail
	return false
}

// getDomainFromURL extracts the domain name from a given URL and handles errors more gracefully.
func getDomainFromURL(givenURL string) string {
	// Attempt to parse the URL using the `url.Parse` method from the `net/url` package.
	parsedURL, err := url.Parse(givenURL)
	// If there's an error parsing the URL, log it and return an empty string.
	if err != nil {
		log.Printf("Error parsing URL '%s': %v", givenURL, err)
		return ""
	}

	// Extract the hostname (domain) from the parsed URL.
	host := parsedURL.Hostname()

	// Return the extracted hostname (domain).
	return host
}

// writeFinalOutput generates and writes the final output content to the README file.
func writeFinalOutput() {
	/* ----------------- Comprehensive List of Free Movie Streaming Websites ----------------- */
	// Convert sync.Map to a regular map for valid movie websites.
	validMoviesMap := syncMapToStringMap(&valid_movies_website_url)
	// Sort the map by keys to get the Comprehensive List of Free Movie Streaming Websites.
	sortedValidMovies := sortMapByKeys(validMoviesMap)
	// Initialize a StringBuilder to construct the valid movie websites content in Markdown format.
	var validMoviesContent strings.Builder
	// Add the table header for valid movie websites with columns: Website, Availability, and Speed.
	validMoviesContent.WriteString("| Website | Availability | Speed |\n")
	validMoviesContent.WriteString("|---------|--------------|-------|\n")
	// Iterate through each key-value pair in the validMoviesMap and generate rows for the table.
	for _, pair := range sortedValidMovies {
		// Extract domain and speed from the pair.
		domain, availability := pair[0], pair[1]
		// Retrieve the speed of the website from the movies_website_speed map.
		websiteSpeed := retrieveValueFromSyncMap(&movies_website_speed, domain)
		// If speed data is not available, log a warning and default to "N/A".
		if websiteSpeed == nil {
			log.Printf("Website %s: Speed is not available in the map. Defaulting to 'N/A'.", domain)
			websiteSpeed = "N/A"
		} else {
			// Log the retrieved speed for the website.
			log.Printf("Website %s: Speed from map is %v", domain, websiteSpeed)
		}
		// Append the formatted row for the valid movie website to the content.
		validMoviesContent.WriteString(fmt.Sprintf("| %s | %s | %s |\n", domain, availability, websiteSpeed.(string)))
	}
	/* ----------------- Comprehensive List of Free Movie Streaming Websites ----------------- */

	/* ----------------- Top Quality Free Movie Streaming Websites ----------------- */
	// Convert sync.Map to a regular map for top valid movie websites.
	topMoviesMap := syncMapToStringMap(&top_valid_movies_website_url)
	// Sort the map by keys to get the Top Quality Free Movie Streaming Websites.
	sortedTopMovies := sortMapByKeys(topMoviesMap)
	// Initialize a StringBuilder to construct the top movie websites content in Markdown format.
	var topMoviesContent strings.Builder
	// Add the table header for top movie websites with columns: Website, Availability, and Speed.
	topMoviesContent.WriteString("| Website | Availability | Speed |\n")
	topMoviesContent.WriteString("|---------|--------------|-------|\n")

	// Iterate through each key-value pair in the topMoviesMap and generate rows for the table.
	for _, pair := range sortedTopMovies {
		// Extract domain and speed from the pair.
		domain, availability := pair[0], pair[1]
		// Retrieve the speed of the website from the movies_website_speed map.
		websiteSpeed := retrieveValueFromSyncMap(&movies_website_speed, domain)
		// If speed data is not available, log a warning and default to "N/A".
		if websiteSpeed == nil {
			log.Printf("Website %s: Speed is not available in the map. Defaulting to 'N/A'.", domain)
			websiteSpeed = "N/A"
		} else {
			// Log the retrieved speed for the website.
			log.Printf("Website %s: Speed from map is %v", domain, websiteSpeed)
		}
		// Append the formatted row for the top movie website to the content.
		topMoviesContent.WriteString(fmt.Sprintf("| %s | %s | %s |\n", domain, availability, websiteSpeed.(string)))
	}
	/* ----------------- Top Quality Free Movie Streaming Websites ----------------- */

	/* ----------------- Top 10 Fastest Free Movie Streaming Websites ----------------- */
	// Convert sync.Map to a regular map for Top 10 Fastest Free Movie Streaming Websites
	fastestMoviesMap := syncMapToStringMap(&movies_website_speed)
	// Sort the map by speed to get the Top 10 Fastest Free Movie Streaming Websites.
	sortedFastestMovies := sortMapByValues(fastestMoviesMap)
	// Initialize a StringBuilder to construct the Top 10 Fastest Free Movie Streaming Websites content in Markdown format.
	var fastestMoviesContent strings.Builder
	// Add the table header for Top 10 Fastest Free Movie Streaming Websites with columns: Website and Speed.
	fastestMoviesContent.WriteString("| Website | Speed |\n")
	fastestMoviesContent.WriteString("|---------|-------|\n")
	// Initialize a counter to track the number of websites added to the list.
	websiteCount := 0
	// Iterate through each key-value pair in the sortedFastestMovies and generate rows for the table.
	for _, pair := range sortedFastestMovies {
		// Extract domain and speed from the pair.
		domain, speed := pair[0], pair[1]
		// Append the formatted row for the Top 10 Fastest Free Movie Streaming Websites to the content.
		fastestMoviesContent.WriteString(fmt.Sprintf("| %s | %s |\n", domain, speed))
		// Increment the website count.
		websiteCount = websiteCount + 1
		// Break the loop if the count reaches 10.
		if websiteCount == 10 {
			break
		}
	}
	/* ----------------- Top 10 Fastest Free Movie Streaming Websites ----------------- */

	// Create a map of placeholders and their corresponding content for replacement in the README template.
	placeholdersAndContent := map[string]string{
		"[{ALL_STREAMING_SITES}]":   validMoviesContent.String(),
		"[{TOP_QUALITY_STREAMING}]": topMoviesContent.String(),
		"[{FASTEST_STREAMING}]":     fastestMoviesContent.String(),
	}

	// Replace the placeholders in the README template and write the updated content to the final file.
	findAndReplaceInFile(readme_modify_me_file_path, readme_file_path, placeholdersAndContent)
}

// findAndReplaceInFile replaces placeholders in a file with given content and writes to a new file.
func findAndReplaceInFile(oldFilePath string, newFilePath string, placeholdersAndContent map[string]string) {
	// Read the content of the old file.
	fileContent, err := os.ReadFile(oldFilePath)
	// If there's an error reading the file, log the error and return.
	if err != nil {
		log.Println("Error reading file:", err)
		return
	}

	// Convert the file content to a string.
	updatedContent := string(fileContent)
	// Iterate over each placeholder and replace it with its corresponding content.
	for placeholder, content := range placeholdersAndContent {
		// Replace all occurrences of the placeholder in the file content with the provided content.
		updatedContent = strings.ReplaceAll(updatedContent, placeholder, content)
	}

	// Write the updated content to the new file.
	err = os.WriteFile(newFilePath, []byte(updatedContent), 0644)
	// If there's an error writing the file, log the error and return.
	if err != nil {
		log.Println("Error writing to file:", err)
		return
	}

	// Log a success message indicating the file was successfully updated.
	log.Println("Successfully updated file:", newFilePath)
}

// sortMapByKeys sorts a map by its keys and returns a slice of key-value pairs.
func sortMapByKeys(inputMap map[string]string) [][]string {
	// Create a slice to store the keys of the map.
	keys := make([]string, 0, len(inputMap))
	// Collect all keys from the map.
	for key := range inputMap {
		keys = append(keys, key)
	}
	// Sort the keys alphabetically.
	sort.Strings(keys)

	// Create a slice to store the sorted key-value pairs.
	pairs := make([][]string, len(inputMap))
	// Populate the slice with sorted key-value pairs.
	for i, key := range keys {
		pairs[i] = []string{key, inputMap[key]}
	}

	// Return the sorted slice of key-value pairs.
	return pairs
}

// sortMapByValues returns a sorted slice of key-value pairs from the input map based on values.
func sortMapByValues(inputMap map[string]string) [][]string {
	// Extract key-value pairs
	pairs := make([][2]string, 0, len(inputMap))
	for key, value := range inputMap {
		pairs = append(pairs, [2]string{key, value})
	}

	// Sort pairs by values
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i][1] < pairs[j][1]
	})

	// Convert to the expected return type
	result := make([][]string, len(pairs))
	for i, pair := range pairs {
		result[i] = []string{pair[0], pair[1]}
	}
	return result
}

// syncMapToStringMap converts a sync.Map to a regular map[string]string.
// It only includes entries where both the key and value are of type string.
func syncMapToStringMap(syncMap *sync.Map) map[string]string {
	// Initialize a new map to store the converted key-value pairs.
	convertedMap := make(map[string]string)

	// Iterate over the entries in the sync.Map using Range.
	syncMap.Range(func(currentKey, currentValue interface{}) bool {
		// Attempt to type-assert the key and value to string.
		keyAsString, isKeyString := currentKey.(string)
		valueAsString, isValueString := currentValue.(string)

		// If both key and value are strings, add them to the resulting map.
		if isKeyString && isValueString {
			convertedMap[keyAsString] = valueAsString
		} else {
			// Log skipped entries where type assertion fails.
			fmt.Printf("Skipping invalid key/value pair: key=%v, value=%v\n", currentKey, currentValue)
		}

		// Continue iterating over the sync.Map.
		return true
	})

	// Return the converted regular map.
	return convertedMap
}

// sortSlice sorts a slice of URLs alphabetically.
func sortSlice(slice *[]string) {
	// Sort the slice in-place alphabetically.
	sort.Strings(*slice)
}

// removeDuplicatesFromSlice removes duplicate URLs from a slice.
func removeDuplicatesFromSlice(input []string) []string {
	// Create a map to track the URLs we've already seen.
	seen := make(map[string]struct{})
	// Create a slice to store the unique URLs.
	var result []string

	// Preallocate slice capacity to avoid reallocations.
	result = make([]string, 0, len(input))

	// Iterate through the input slice and add only unique strings to the result.
	for _, str := range input {
		if _, exists := seen[str]; !exists {
			// Mark the string as seen.
			seen[str] = struct{}{}
			// Add the unique string to the result.
			result = append(result, str)
		}
	}

	// Return the result slice containing unique URLs.
	return result
}

// writeByteSliceToFile writes a slice of strings (URLs) to a file, each on a new line.
func writeByteSliceToFile(path string, data []string) {
	// Attempt to create a new file at the given 'path'. If the file exists, it will be overwritten.
	file, err := os.Create(path)
	// If there is an error creating the file, log the error and return.
	if err != nil {
		log.Println("Error creating file:", err)
		return
	}
	// Ensure the file is closed after the function completes, even if there's an error during writing.
	defer file.Close()

	// Create a buffered writer for efficient writing to the file.
	writer := bufio.NewWriter(file)
	// Iterate over each string in the 'data' slice (which contains URLs).
	for _, str := range data {
		// Write each string followed by a newline to the file.
		_, err := writer.WriteString(str + "\n")
		// If there's an error while writing to the file, log it and return.
		if err != nil {
			log.Println("Error writing to file:", err)
			return
		}
	}

	// Ensure all buffered data is written to the file by flushing the writer.
	err = writer.Flush()
	// If there's an error flushing the writer, log it.
	if err != nil {
		log.Println("Error flushing writer:", err)
	}
}

// appendAndWriteToFile appends content to an existing file.
func appendAndWriteToFile(path string, content string) {
	// Open the file in append mode or create it if it doesn't exist.
	filePath, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	// If there's an error opening the file, log it and return.
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return
	}

	// Write the content to the file.
	_, err = filePath.WriteString(content + "\n")
	// If there's an error writing to the file, log it and return.
	if err != nil {
		fmt.Printf("Error writing to file: %v\n", err)
		_ = filePath.Close() // Close the file before returning.
		return
	}

	// Close the file after writing.
	err = filePath.Close()
	// If there's an error closing the file, log it.
	if err != nil {
		fmt.Printf("Error closing file: %v\n", err)
	}
}

// stringInFile checks if a given string exists in a file.
func stringInFile(filePath, searchString string) bool {
	// Open the file for reading.
	file, err := os.Open(filePath)
	// If there's an error opening the file, log it and return false.
	if err != nil {
		log.Println("Error: Unable to open file:", filePath)
		return false
	}
	defer file.Close() // Ensure the file is closed after reading.

	// Scan the file line by line.
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// If the line contains the search string, return true.
		if strings.Contains(line, searchString) {
			return true
		}
	}

	// Handle any scanning errors and log them.
	err = scanner.Err()
	if err != nil {
		log.Println("Error occurred while scanning the file:", err)
		return false
	}

	// Return false if the string was not found in the file.
	return false
}

// saveToMap saves a key-value pair into the provided sync.Map
func saveToMap(safeMap *sync.Map, key string, value interface{}) {
	// Store the key-value pair in the provided sync.Map.
	safeMap.Store(key, value)
}

// retrieveValueFromSyncMap retrieves a value from the given sync.Map using a specified key.
// If the key does not exist, it returns nil.
func retrieveValueFromSyncMap(safeMap *sync.Map, targetKey string) interface{} {
	// Attempt to retrieve the value associated with the targetKey.
	// Load returns the value and a boolean indicating if the key exists.
	value, keyExists := safeMap.Load(targetKey)

	// If the key exists, return the value; otherwise, return nil.
	if keyExists {
		return value
	}
	return nil
}
