package main

import (
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

// Version information
var (
	AppName    = "Local File Server"
	AppVersion = "1.0.0" // This will be overridden during build with -X flag
)

// Configuration for the file server
type Config struct {
	Port        int
	DownloadDir string
	LocalOnly   bool
	ShowVersion bool
	ShowHelp    bool
}

// Usage information for the program
func printUsage() {
	fmt.Printf("%s v%s\n", AppName, AppVersion)
	fmt.Println("A simple HTTP file server for sharing files on your local network")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  local-fileserver [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -port int")
	fmt.Println("        Port to serve on (default 8080)")
	fmt.Println("  -dir string")
	fmt.Println("        Directory to serve files from (default is ~/Downloads)")
	fmt.Println("  -local")
	fmt.Println("        Restrict access to local network only (default true)")
	fmt.Println("  -version")
	fmt.Println("        Show version information")
	fmt.Println("  -help")
	fmt.Println("        Show this help message")
	fmt.Println()
	fmt.Println("Example:")
	fmt.Println("  local-fileserver -port 9000 -dir /path/to/files -local=false")
	fmt.Println()
}

// Template for the file listing and upload page
const htmlTemplate = `
<!DOCTYPE html>
<html>
<head>
    <title>Local File Server</title>
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>
        body {
            font-family: Arial, sans-serif;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
        }
        h1 {
            color: #333;
        }
        .file {
            margin: 5px 0;
            padding: 8px;
            background-color: #f5f5f5;
            border-radius: 4px;
        }
        .file a {
            text-decoration: none;
            color: #0066cc;
        }
        .file a:hover {
            text-decoration: underline;
        }
        .folder {
            margin: 5px 0;
            padding: 8px;
            background-color: #e1f5fe;
            border-radius: 4px;
            cursor: pointer;
        }
        .folder-name {
            font-weight: bold;
            color: #0277bd;
        }
        .folder-icon:before {
            content: "📁 ";
        }
        .folder-expanded .folder-icon:before {
            content: "📂 ";
        }
        .children {
            margin-left: 20px;
            border-left: 1px solid #ccc;
            padding-left: 10px;
        }
        .upload-form {
            margin: 20px 0;
            padding: 15px;
            background-color: #e9e9e9;
            border-radius: 5px;
        }
        .upload-button {
            margin-top: 10px;
            padding: 8px 16px;
            background-color: #4CAF50;
            color: white;
            border: none;
            border-radius: 4px;
            cursor: pointer;
        }
        .upload-button:hover {
            background-color: #45a049;
        }
        .breadcrumb {
            margin-bottom: 15px;
            padding: 8px;
            background-color: #f0f0f0;
            border-radius: 4px;
        }
        .breadcrumb a {
            text-decoration: none;
            color: #0066cc;
        }
        .breadcrumb a:hover {
            text-decoration: underline;
        }
        .search-container {
            margin: 15px 0;
            display: flex;
            align-items: center;
        }
        .search-input {
            flex: 1;
            padding: 8px 12px;
            border: 1px solid #ccc;
            border-radius: 4px;
            font-size: 14px;
        }
        .search-input:focus {
            border-color: #0066cc;
            outline: none;
        }
        .clear-search {
            margin-left: 8px;
            padding: 8px 12px;
            background-color: #f0f0f0;
            border: none;
            border-radius: 4px;
            cursor: pointer;
            font-size: 14px;
        }
        .clear-search:hover {
            background-color: #e0e0e0;
        }
        .hidden {
            display: none !important;
        }
        .folder-actions {
            margin: 15px 0;
            display: flex;
            justify-content: flex-start;
        }
        .toggle-folders-button {
            padding: 8px 16px;
            background-color: #0277bd;
            color: white;
            border: none;
            border-radius: 4px;
            cursor: pointer;
            font-size: 14px;
        }
        .toggle-folders-button:hover {
            background-color: #015384;
        }
    </style>
    <script>
        function toggleFolder(path, event) {
            // Stop event propagation to prevent parent folders from toggling
            if (event) {
                event.stopPropagation();
            }
            
            const folder = document.getElementById('folder-' + path);
            const children = document.getElementById('children-' + path);
            
            if (children.style.display === 'none') {
                children.style.display = 'block';
                folder.classList.add('folder-expanded');
            } else {
                children.style.display = 'none';
                folder.classList.remove('folder-expanded');
            }
        }

        // Function to filter files and folders as user types
        function filterFileList() {
            const searchTerm = document.getElementById('search-input').value.toLowerCase().trim();
            const fileElements = document.querySelectorAll('.file');
            const folderElements = document.querySelectorAll('.folder');
            const noResultsMessage = document.getElementById('no-search-results');
            const toggleButton = document.getElementById('toggle-folders-button');
            
            let visibleItems = 0;
            let expandedFolders = 0;
            let totalFolders = 0;
            
            // Function to check if text contains search term
            const matchesSearch = (text) => text.toLowerCase().includes(searchTerm);
            
            // Filter files
            fileElements.forEach(file => {
                const fileName = file.querySelector('a').textContent;
                const isMatch = searchTerm === '' || matchesSearch(fileName);
                file.classList.toggle('hidden', !isMatch);
                if (isMatch) visibleItems++;
            });
            
            // Filter folders and their children
            folderElements.forEach(folder => {
                totalFolders++;
                const folderName = folder.querySelector('.folder-name').textContent;
                const isMatch = searchTerm === '' || matchesSearch(folderName);
                const childrenContainer = document.getElementById('children-' + folder.id.substring(7)); // Remove 'folder-' prefix
                
                // Check if any children are visible when searching
                let hasVisibleChildren = false;
                if (childrenContainer) {
                    const childFiles = childrenContainer.querySelectorAll('.file');
                    const childFolders = childrenContainer.querySelectorAll('.folder');
                    
                    // Check child files
                    childFiles.forEach(childFile => {
                        const childFileName = childFile.querySelector('a').textContent;
                        const childMatch = searchTerm === '' || matchesSearch(childFileName);
                        childFile.classList.toggle('hidden', !childMatch);
                        hasVisibleChildren = hasVisibleChildren || childMatch;
                    });
                    
                    // Check child folders
                    childFolders.forEach(childFolder => {
                        const childFolderName = childFolder.querySelector('.folder-name').textContent;
                        const childMatch = searchTerm === '' || matchesSearch(childFolderName);
                        hasVisibleChildren = hasVisibleChildren || childMatch;
                    });
                }
                
                // Show folder if it matches search or has matching children
                folder.classList.toggle('hidden', !isMatch && !hasVisibleChildren);
                
                // Expand folder if we're searching and there are matches inside
                if (searchTerm !== '' && hasVisibleChildren) {
                    childrenContainer.style.display = 'block';
                    folder.classList.add('folder-expanded');
                    expandedFolders++;
                } else if (searchTerm === '') {
                    // Restore collapsed state when search is cleared
                    childrenContainer.style.display = 'none';
                    folder.classList.remove('folder-expanded');
                } else if (childrenContainer.style.display === 'block') {
                    // Count already expanded folders
                    expandedFolders++;
                }
                
                if (isMatch || hasVisibleChildren) visibleItems++;
            });
            
            // Update the global state and button text based on the actual state of folders
            if (totalFolders > 0) {
                // Update allFoldersExpanded based on if all folders are expanded
                allFoldersExpanded = (expandedFolders === totalFolders);
                
                // Update button text to match current state
                if (toggleButton) {
                    toggleButton.textContent = allFoldersExpanded ? 'Collapse All Folders' : 'Expand All Folders';
                }
            }
            
            // Show a message if no results found
            if (noResultsMessage) {
                noResultsMessage.style.display = visibleItems > 0 ? 'none' : 'block';
            }
        }
        
        function clearSearch() {
            const searchInput = document.getElementById('search-input');
            searchInput.value = '';
            filterFileList();
            searchInput.focus();
        }
        
        // Initialize search when the page loads
        document.addEventListener('DOMContentLoaded', function() {
            const searchInput = document.getElementById('search-input');
            if (searchInput) {
                searchInput.addEventListener('input', filterFileList);
                searchInput.addEventListener('keydown', function(e) {
                    // Clear search on Escape key
                    if (e.key === 'Escape') {
                        clearSearch();
                    }
                });
            }
            
            const clearButton = document.getElementById('clear-search');
            if (clearButton) {
                clearButton.addEventListener('click', clearSearch);
            }
            
            // Set up expand/collapse button functionality
            const toggleFoldersButton = document.getElementById('toggle-folders-button');
            if (toggleFoldersButton) {
                toggleFoldersButton.addEventListener('click', toggleAllFolders);
            }
        });
        
        // Global variable to track current folder expansion state
        let allFoldersExpanded = false;
        
        // Function to toggle all folders
        function toggleAllFolders() {
            const folderElements = document.querySelectorAll('.folder');
            const toggleButton = document.getElementById('toggle-folders-button');
            
            // Toggle the global state
            allFoldersExpanded = !allFoldersExpanded;
            
            // Update button text
            if (toggleButton) {
                toggleButton.textContent = allFoldersExpanded ? 'Collapse All Folders' : 'Expand All Folders';
            }
            
            // For each folder, expand or collapse based on new state
            folderElements.forEach(folder => {
                const folderId = folder.id;
                const folderPath = folderId.substring(7); // Remove 'folder-' prefix
                const childrenContainer = document.getElementById('children-' + folderPath);
                
                if (childrenContainer) {
                    childrenContainer.style.display = allFoldersExpanded ? 'block' : 'none';
                    
                    if (allFoldersExpanded) {
                        folder.classList.add('folder-expanded');
                    } else {
                        folder.classList.remove('folder-expanded');
                    }
                }
            });
        }
    </script>
</head>
<body>
    <h1>Local File Server</h1>
    
    <div class="upload-form">
        <h3>Upload File</h3>
        <form method="post" enctype="multipart/form-data">
            <input type="file" name="file" required>
            <input type="hidden" name="path" value="{{.CurrentPath}}">
            <br>
            <button type="submit" class="upload-button">Upload</button>
        </form>
    </div>

    {{if .CurrentPath}}
    <div class="breadcrumb">
        <a href="/?path=">Home</a>
        {{range $index, $part := .Breadcrumbs}}
            / <a href="/?path={{$part.Path}}">{{$part.Name}}</a>
        {{end}}
    </div>
    {{end}}

    <h3>Files and Folders</h3>
    
    <div class="search-container">
        <input type="text" id="search-input" class="search-input" placeholder="Search files and folders..." autocomplete="off">
        <button id="clear-search" class="clear-search" title="Clear search">✕</button>
    </div>
    
    <div id="no-search-results" style="display: none;">
        <p>No files or folders match your search.</p>
    </div>
    
    <div class="folder-actions">
        <button id="toggle-folders-button" class="toggle-folders-button">Expand All Folders</button>
    </div>
    
    {{define "file_item"}}
        {{if .IsDir}}
            <div id="folder-{{.Path}}" class="folder" onclick="toggleFolder('{{.Path}}', event)">
                <span class="folder-icon"></span>
                <a href="/?path={{.Path}}" class="folder-name">{{.Name}}</a>
            </div>
            <div id="children-{{.Path}}" class="children" style="display: {{if .Expanded}}block{{else}}none{{end}};">
                {{range .Children}}
                    {{template "file_item" .}}
                {{end}}
            </div>
        {{else}}
            <div class="file">
                <a href="/download/{{.Path}}">{{.Name}}</a> ({{.Size}} bytes)
            </div>
        {{end}}
    {{end}}
    
    {{range .Files}}
        {{template "file_item" .}}
    {{else}}
        <p>No files found</p>
    {{end}}
</body>
</html>
`

// FileInfo represents a file or directory in the downloads directory
type FileInfo struct {
	Name     string
	Size     int64
	IsDir    bool
	Path     string
	Children []FileInfo
	Expanded bool
}

// BreadcrumbItem represents a path segment for navigation
type BreadcrumbItem struct {
	Name string
	Path string
}

// Safely join and clean a path, ensuring it doesn't escape the base directory
func safeJoinPath(baseDir, userPath string) (string, error) {
	// Clean the path to remove any ".." elements
	cleanedPath := filepath.Clean(userPath)

	// Remove leading slash or backslash if any
	cleanedPath = strings.TrimPrefix(cleanedPath, "/")
	cleanedPath = strings.TrimPrefix(cleanedPath, "\\")

	// Join with the base directory
	fullPath := filepath.Join(baseDir, cleanedPath)

	// Ensure the path is still within the base directory
	relPath, err := filepath.Rel(baseDir, fullPath)
	if err != nil {
		return "", err
	}

	// Check if the resulting path tries to go outside the base directory
	if strings.HasPrefix(relPath, "..") {
		return "", fmt.Errorf("path escapes the base directory")
	}

	return fullPath, nil
}

// Generate breadcrumb items for navigation
func generateBreadcrumbs(path string) []BreadcrumbItem {
	if path == "" {
		return []BreadcrumbItem{}
	}

	parts := strings.Split(path, "/")
	breadcrumbs := make([]BreadcrumbItem, len(parts))

	currentPath := ""
	for i, part := range parts {
		if part == "" {
			continue
		}

		if i > 0 {
			currentPath += "/"
		}
		currentPath += part

		breadcrumbs[i] = BreadcrumbItem{
			Name: part,
			Path: currentPath,
		}
	}

	// Remove empty items
	result := []BreadcrumbItem{}
	for _, b := range breadcrumbs {
		if b.Name != "" {
			result = append(result, b)
		}
	}

	return result
}

// List files and directories with their children recursively up to a specified depth
func listFilesRecursive(baseDir, relativePath string, depth int) ([]FileInfo, error) {
	currentPath, err := safeJoinPath(baseDir, relativePath)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(currentPath)
	if err != nil {
		return nil, err
	}

	result := make([]FileInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		entryPath := filepath.Join(relativePath, entry.Name())
		if entryPath == "" {
			entryPath = entry.Name()
		}

		fileInfo := FileInfo{
			Name:     entry.Name(),
			Size:     info.Size(),
			IsDir:    entry.IsDir(),
			Path:     entryPath,
			Expanded: false,
			Children: []FileInfo{},
		}

		// If it's a directory and we haven't reached the max depth, get its children
		if entry.IsDir() && depth > 0 {
			children, err := listFilesRecursive(baseDir, entryPath, depth-1)
			if err == nil {
				fileInfo.Children = children
			}
		}

		result = append(result, fileInfo)
	}

	return result, nil
}

// Check if an IP address belongs to the local network
func isLocalIP(addr string) bool {
	ip := net.ParseIP(addr)
	if ip == nil {
		return false
	}

	// Localhost
	if ip.IsLoopback() {
		log.Printf("IP %v is loopback", ip)
		return true
	}

	// Check for private IP ranges using CIDR notation
	privateIPBlocks := []*net.IPNet{
		mustParseCIDR("10.0.0.0/8"),     // RFC1918
		mustParseCIDR("172.16.0.0/12"),  // RFC1918
		mustParseCIDR("192.168.0.0/16"), // RFC1918
		mustParseCIDR("169.254.0.0/16"), // RFC3927 (Link-local)
		mustParseCIDR("fd00::/8"),       // RFC4193 (Unique local IPv6)
	}

	for _, block := range privateIPBlocks {
		if block.Contains(ip) {
			log.Printf("IP %v is in private range %v", ip, block)
			return true
		}
	}

	return false
}

// Helper function to parse CIDR strings
func mustParseCIDR(s string) *net.IPNet {
	_, block, err := net.ParseCIDR(s)
	if err != nil {
		panic(fmt.Errorf("invalid CIDR block: %q", s))
	}
	return block
}

// Local network filtering middleware
func localNetworkFilter(next http.Handler, localOnly bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if localOnly {
			// Get client IP by stripping port number if present
			clientIP := r.RemoteAddr
			if i := strings.LastIndex(clientIP, ":"); i != -1 {
				clientIP = clientIP[:i]
			}

			if !isLocalIP(clientIP) {
				http.Error(w, "Access denied: only local network connections are allowed", http.StatusForbidden)
				log.Printf("Blocked access from non-local IP: %s", clientIP)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	// Set up configuration with flags
	config := Config{}

	// Get user's home directory
	usr, err := user.Current()
	if err != nil {
		log.Fatalf("Error getting user's home directory: %v", err)
	}

	// Default to Downloads folder in the user's home directory
	downloadsDir := filepath.Join(usr.HomeDir, "Downloads")

	flag.IntVar(&config.Port, "port", 8080, "Port to serve on")
	flag.StringVar(&config.DownloadDir, "dir", downloadsDir, "Directory to serve files from")
	flag.BoolVar(&config.LocalOnly, "local", true, "Restrict access to local network only")
	flag.BoolVar(&config.ShowVersion, "version", false, "Show version information")
	flag.BoolVar(&config.ShowHelp, "help", false, "Show this help message")
	flag.Parse()

	// Show version information and exit if requested
	if config.ShowVersion {
		fmt.Printf("%s v%s\n", AppName, AppVersion)
		return
	}

	// Show usage information and exit if requested
	if config.ShowHelp {
		printUsage()
		return
	}

	// Ensure the download directory exists
	if _, err := os.Stat(config.DownloadDir); os.IsNotExist(err) {
		log.Fatalf("Download directory does not exist: %s", config.DownloadDir)
	}

	// Parse the HTML template
	tmpl, err := template.New("fileList").Parse(htmlTemplate)
	if err != nil {
		log.Fatalf("Error parsing template: %v", err)
	}

	// Set up handlers

	// Handler for the home page (file listing and upload form)
	homeHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		// Get the requested path from query parameter
		requestedPath := r.URL.Query().Get("path")
		// Clean and validate the path
		requestedPath = strings.TrimPrefix(requestedPath, "/")

		if r.Method == "POST" {
			// Handle file upload
			file, header, err := r.FormFile("file")
			if err != nil {
				http.Error(w, "Error retrieving file from form: "+err.Error(), http.StatusBadRequest)
				return
			}
			defer file.Close()

			// Get the target path for uploading
			targetPath := r.FormValue("path")

			// Create the target directory if it doesn't exist yet
			uploadDir, err := safeJoinPath(config.DownloadDir, targetPath)
			if err != nil {
				http.Error(w, "Invalid upload path: "+err.Error(), http.StatusBadRequest)
				return
			}

			// Make sure the directory exists
			err = os.MkdirAll(uploadDir, 0755)
			if err != nil {
				http.Error(w, "Error creating directory: "+err.Error(), http.StatusInternalServerError)
				return
			}

			// Create a new file in the target directory
			filename := filepath.Join(uploadDir, header.Filename)
			out, err := os.Create(filename)
			if err != nil {
				http.Error(w, "Error creating file: "+err.Error(), http.StatusInternalServerError)
				return
			}
			defer out.Close()

			// Copy the uploaded file to the destination file
			_, err = io.Copy(out, file)
			if err != nil {
				http.Error(w, "Error saving file: "+err.Error(), http.StatusInternalServerError)
				return
			}

			log.Printf("File uploaded successfully: %s to %s", header.Filename, targetPath)

			// Redirect back to the same path
			redirectURL := "/"
			if targetPath != "" {
				redirectURL += "?path=" + targetPath
			}
			http.Redirect(w, r, redirectURL, http.StatusSeeOther)
			return
		}

		// For GET requests, list files and directories
		files, err := listFilesRecursive(config.DownloadDir, requestedPath, 10)
		if err != nil {
			http.Error(w, "Error reading directory: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Generate breadcrumbs for navigation
		breadcrumbs := generateBreadcrumbs(requestedPath)

		// Render the template
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		err = tmpl.Execute(w, struct {
			Files       []FileInfo
			CurrentPath string
			Breadcrumbs []BreadcrumbItem
		}{
			Files:       files,
			CurrentPath: requestedPath,
			Breadcrumbs: breadcrumbs,
		})

		if err != nil {
			http.Error(w, "Error rendering page: "+err.Error(), http.StatusInternalServerError)
			return
		}
	})

	// Handler for downloading files
	downloadHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		filePath := strings.TrimPrefix(r.URL.Path, "/download/")
		if filePath == "" {
			http.Error(w, "No file specified", http.StatusBadRequest)
			return
		}

		// Get the full path in a safe way, preventing directory traversal
		fullPath, err := safeJoinPath(config.DownloadDir, filePath)
		if err != nil {
			http.Error(w, "Invalid file path: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Check if the file exists
		fileInfo, err := os.Stat(fullPath)
		if err != nil {
			if os.IsNotExist(err) {
				http.Error(w, "File not found", http.StatusNotFound)
			} else {
				http.Error(w, "Error accessing file: "+err.Error(), http.StatusInternalServerError)
			}
			return
		}

		// Check if it's a regular file
		if fileInfo.IsDir() {
			http.Error(w, "Cannot download directories", http.StatusBadRequest)
			return
		}

		// Set headers for file download
		filename := filepath.Base(filePath)
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

		// Serve the file
		http.ServeFile(w, r, fullPath)
		log.Printf("File downloaded: %s", filePath)
	})

	// Set up the server with local network filtering
	mux := http.NewServeMux()
	mux.Handle("/", localNetworkFilter(homeHandler, config.LocalOnly))
	mux.Handle("/download/", localNetworkFilter(downloadHandler, config.LocalOnly))

	// Get the IP address of this machine to display in the startup message
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Printf("Error getting network interfaces: %v", err)
	}

	log.Printf("Starting file server on port %d", config.Port)
	log.Printf("Serving files from: %s", config.DownloadDir)
	log.Printf("Local network access only: %v", config.LocalOnly)

	// Print potential URLs to access the server
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			log.Printf("Access the server at: http://%s:%d", ipnet.IP.String(), config.Port)
		}
	}

	// Always show localhost as an option
	log.Printf("Access the server at: http://localhost:%d", config.Port)

	// Start the server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.Port),
		Handler: mux,
	}

	log.Fatal(server.ListenAndServe())
}
