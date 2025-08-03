// API Base URL - backend runs on port 8080
const API_BASE_URL = 'http://localhost:8080';

// Global state
let currentPage = 1;
let currentQuery = '';
let totalPages = 1;
let isLoading = false;
let recentPaths = JSON.parse(localStorage.getItem('recentPaths')) || [];
let recentSearches = JSON.parse(localStorage.getItem('recentSearches')) || [];
let currentIndexRequestID = null;
let pollingInterval = null;

// DOM Elements
const folderPathInput = document.getElementById('folder-path');
const suggestionsBtn = document.getElementById('suggestions-btn');
const suggestionsDropdown = document.getElementById('suggestions-dropdown');
const recentPathsSection = document.getElementById('recent-paths-section');
const recentPathsContainer = document.getElementById('recent-paths-container');
const indexBtn = document.getElementById('index-btn');
const indexStatus = document.getElementById('index-status');
const indexProgressContainer = document.getElementById('index-progress-container');
const indexProgressBar = document.getElementById('index-progress-bar');
const indexProgressText = document.getElementById('index-progress-text');

const searchQueryInput = document.getElementById('search-query');
const recentSearchesBtn = document.getElementById('recent-searches-btn');
const recentSearchesDropdown = document.getElementById('recent-searches-dropdown');
const recentSearchesSection = document.getElementById('recent-searches-section');
const recentSearchesContainer = document.getElementById('recent-searches-container');
const searchBtn = document.getElementById('search-btn');
const searchStatus = document.getElementById('search-status');

const resultsContainer = document.getElementById('results-container');
const resultsSection = document.querySelector('.results-section');
const pagination = document.getElementById('pagination');
const prevBtn = document.getElementById('prev-btn');
const nextBtn = document.getElementById('next-btn');
const pageInfo = document.getElementById('page-info');

// Event listeners
document.addEventListener('DOMContentLoaded', function() {
    indexBtn.addEventListener('click', handleIndex);
    searchBtn.addEventListener('click', () => handleSearch());
    prevBtn.addEventListener('click', (e) => handlePrevPage(e));
    nextBtn.addEventListener('click', (e) => handleNextPage(e));
    suggestionsBtn.addEventListener('click', toggleSuggestions);
    recentSearchesBtn.addEventListener('click', toggleRecentSearches);
    
    // Setup suggestion items
    setupSuggestionItems();
    updateRecentPaths();
    updateRecentSearches();
    
    // Close dropdown when clicking outside
    document.addEventListener('click', function(e) {
        if (!suggestionsDropdown.contains(e.target) && !suggestionsBtn.contains(e.target)) {
            suggestionsDropdown.classList.remove('active');
        }
        if (!recentSearchesDropdown.contains(e.target) && !recentSearchesBtn.contains(e.target)) {
            recentSearchesDropdown.classList.remove('active');
        }
    });
    
    // Enter key support for search
    searchQueryInput.addEventListener('keypress', function(e) {
        if (e.key === 'Enter') handleSearch();
    });
    
    // Enter key support for folder path
    folderPathInput.addEventListener('keypress', function(e) {
        if (e.key === 'Enter') handleIndex();
    });
});

// Status message utility
function showStatus(element, message, type = 'loading') {
    element.textContent = message;
    element.className = `status-message ${type}`;
    element.style.display = 'block';
}

function hideStatus(element) {
    element.style.display = 'none';
}

// Toggle suggestions dropdown
function toggleSuggestions() {
    suggestionsDropdown.classList.toggle('active');
}

// Setup suggestion items
function setupSuggestionItems() {
    const suggestionItems = document.querySelectorAll('.suggestion-item');
    suggestionItems.forEach(item => {
        item.addEventListener('click', function() {
            const path = this.getAttribute('data-path');
            folderPathInput.value = path;
            suggestionsDropdown.classList.remove('active');
        });
    });
}

// Update recent paths
function updateRecentPaths() {
    if (recentPaths.length > 0) {
        recentPathsSection.style.display = 'block';
        recentPathsContainer.innerHTML = '';
        
        recentPaths.forEach(path => {
            const item = document.createElement('div');
            item.className = 'suggestion-item';
            item.setAttribute('data-path', path);
            item.textContent = path.split('/').pop() || path; // Show folder name
            item.addEventListener('click', function() {
                folderPathInput.value = path;
                suggestionsDropdown.classList.remove('active');
            });
            recentPathsContainer.appendChild(item);
        });
    } else {
        recentPathsSection.style.display = 'none';
    }
}

// Add to recent paths
function addToRecentPaths(path) {
    if (!recentPaths.includes(path)) {
        recentPaths.unshift(path);
        if (recentPaths.length > 5) {
            recentPaths.pop();
        }
        localStorage.setItem('recentPaths', JSON.stringify(recentPaths));
        updateRecentPaths();
    }
}

// Toggle recent searches dropdown
function toggleRecentSearches() {
    recentSearchesDropdown.classList.toggle('active');
}

// Update recent searches
function updateRecentSearches() {
    if (recentSearches.length > 0) {
        recentSearchesSection.style.display = 'block';
        recentSearchesContainer.innerHTML = '';
        
        recentSearches.forEach(search => {
            const item = document.createElement('div');
            item.className = 'suggestion-item';
            item.textContent = search;
            item.addEventListener('click', function() {
                searchQueryInput.value = search;
                recentSearchesDropdown.classList.remove('active');
            });
            recentSearchesContainer.appendChild(item);
        });
    } else {
        recentSearchesSection.style.display = 'none';
    }
}

// Add to recent searches
function addToRecentSearches(query) {
    if (query && !recentSearches.includes(query)) {
        recentSearches.unshift(query);
        if (recentSearches.length > 5) {
            recentSearches.pop();
        }
        localStorage.setItem('recentSearches', JSON.stringify(recentSearches));
        updateRecentSearches();
    }
}

// Index functionality
async function handleIndex() {
    const folderPath = folderPathInput.value.trim();
    
    if (!folderPath) {
        showStatus(indexStatus, 'Please enter a folder path', 'error');
        return;
    }
    
    if (isLoading) return;
    
    isLoading = true;
    indexBtn.disabled = true;
    hideStatus(indexStatus);
    showProgressBar();
    updateProgress(0);
    
    try {
        const response = await fetch(`${API_BASE_URL}/index`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                path: folderPath
            })
        });
        
        if (response.ok) {
            const data = await response.json();
            currentIndexRequestID = data.data.request_id;
            
            // Start polling for progress
            startPolling(folderPath);
        } else {
            const data = await response.json();
            hideProgressBar();
            showStatus(indexStatus, `Error: ${data.errors ? data.errors.join(', ') : 'Failed to start indexing'}`, 'error');
            isLoading = false;
            indexBtn.disabled = false;
        }
    } catch (error) {
        hideProgressBar();
        showStatus(indexStatus, `Error: ${error.message}`, 'error');
        isLoading = false;
        indexBtn.disabled = false;
    }
}

// Search functionality
async function handleSearch(page = 1) {
    const query = searchQueryInput.value.trim();
    
    if (!query) {
        showStatus(searchStatus, 'Please enter a search query', 'error');
        return;
    }
    
    if (isLoading) return;
    
    isLoading = true;
    searchBtn.disabled = true;
    showStatus(searchStatus, 'Searching...', 'loading');
    
    try {
        const response = await fetch(`${API_BASE_URL}/search?query=${encodeURIComponent(query)}&page=${page}&per_page=10`);
        const data = await response.json();
        
        if (response.ok) {
            currentQuery = query;
            currentPage = page;
            displayResults(data);
            hideStatus(searchStatus);
            // Add to recent searches only on successful search
            addToRecentSearches(query);
        } else {
            showStatus(searchStatus, `Error: ${data.errors ? data.errors.join(', ') : 'Search failed'}`, 'error');
        }
    } catch (error) {
        showStatus(searchStatus, `Error: ${error.message}`, 'error');
    } finally {
        isLoading = false;
        searchBtn.disabled = false;
    }
}

// Display search results
function displayResults(data) {
    // Access the nested data structure
    const results = data.data?.results || [];
    const pageDetails = data.data?.page_details || {};
    
    totalPages = pageDetails.total_pages || 1;
    currentPage = pageDetails.current_page || 1;
    
    resultsContainer.innerHTML = '';
    
    if (results.length === 0) {
        resultsContainer.innerHTML = '<p class="no-results">No results found for your search query.</p>';
    } else {
        results.forEach(result => {
            const resultElement = createResultElement(result);
            resultsContainer.appendChild(resultElement);
        });
    }
    
    updatePagination();
    resultsSection.classList.add('visible');
}

// Create individual result element
function createResultElement(result) {
    const resultDiv = document.createElement('div');
    resultDiv.className = 'result-item';
    
    // Header with file name and open button
    const headerDiv = document.createElement('div');
    headerDiv.className = 'result-header';
    
    const nameDiv = document.createElement('div');
    nameDiv.className = 'result-name';
    nameDiv.textContent = result.name || 'Unknown file';
    
    const copyBtn = document.createElement('button');
    copyBtn.className = 'copy-path-btn'; // Updated class name for styling
    copyBtn.textContent = 'Copy Path';
    // Pass the button element itself to the handler for feedback
    copyBtn.onclick = (e) => copyFilePath(result.path, e.target);
    
    headerDiv.appendChild(nameDiv);
    headerDiv.appendChild(copyBtn);
    
    // Path display
    const pathDiv = document.createElement('div');
    pathDiv.className = 'result-path';
    pathDiv.textContent = result.path || 'Unknown path';
    
    // File info
    const infoDiv = document.createElement('div');
    infoDiv.className = 'result-info';
    infoDiv.innerHTML = `
        <span class="result-size">Size: ${formatFileSize(result.size || 0)}</span>
        ${result.mod_time ? `<span class="result-time">Modified: ${result.mod_time}</span>` : ''}
    `;
    
    resultDiv.appendChild(headerDiv);
    resultDiv.appendChild(pathDiv);
    resultDiv.appendChild(infoDiv);
    
    // Add snippet if available
    if (result.snippet && result.snippet.trim() !== '') {
        const snippetDiv = document.createElement('div');
        snippetDiv.className = 'result-snippet';
        snippetDiv.textContent = result.snippet;
        resultDiv.appendChild(snippetDiv);
    }
    
    return resultDiv;
}

function copyFilePath(filePath, buttonElement) {
    navigator.clipboard.writeText(filePath).then(() => {
        // Save the original text and disable the button
        const originalText = buttonElement.textContent;
        buttonElement.textContent = 'Copied!';
        buttonElement.disabled = true;

        // Change the text back after 2 seconds
        setTimeout(() => {
            buttonElement.textContent = originalText;
            buttonElement.disabled = false;
        }, 2000);
    }).catch(err => {
        console.error('Failed to copy: ', err);
        alert('Failed to copy path automatically. Please check browser permissions.');
    });
}

// Format file size function
function formatFileSize(bytes) {
    if (bytes === 0) return '0 B';
    
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

// Update pagination controls
function updatePagination() {
    prevBtn.disabled = currentPage <= 1;
    nextBtn.disabled = currentPage >= totalPages;
    pageInfo.textContent = `Page ${currentPage} of ${totalPages}`;
}

// Pagination handlers
function handlePrevPage() {
    if (currentPage > 1) {
        handleSearch(currentPage - 1);
    }
}

function handleNextPage() {
    if (currentPage < totalPages) {
        handleSearch(currentPage + 1);
    }
}

// Progress bar functions
function showProgressBar() {
    indexProgressContainer.style.display = 'block';
}

function hideProgressBar() {
    indexProgressContainer.style.display = 'none';
}

function updateProgress(percentage) {
    indexProgressBar.style.width = `${percentage}%`;
    indexProgressText.textContent = `${percentage}%`;
}

// Polling functions
function startPolling(folderPath) {
    if (pollingInterval) {
        clearInterval(pollingInterval);
    }
    
    pollingInterval = setInterval(async () => {
        try {
            const response = await fetch(`${API_BASE_URL}/index/${currentIndexRequestID}`);
            const data = await response.json();
            const status = data.data.status;
            
            if (response.ok && status >= 0) {
                updateProgress(status);
                
                if (status >= 100) {
                    // Indexing complete
                    stopPolling();
                    hideProgressBar();
                    showStatus(indexStatus, `Successfully indexed files from ${folderPath}`, 'success');
                    addToRecentPaths(folderPath);
                    isLoading = false;
                    indexBtn.disabled = false;
                    currentIndexRequestID = null;
                }
            } else {
                // Handle error case
                stopPolling();
                hideProgressBar();
                showStatus(indexStatus, 'Error checking indexing progress', 'error');
                isLoading = false;
                indexBtn.disabled = false;
                currentIndexRequestID = null;
            }
        } catch (error) {
            stopPolling();
            hideProgressBar();
            showStatus(indexStatus, `Error: ${error.message}`, 'error');
            isLoading = false;
            indexBtn.disabled = false;
            currentIndexRequestID = null;
        }
    }, 3000); // Poll every 3 seconds
}

function stopPolling() {
    if (pollingInterval) {
        clearInterval(pollingInterval);
        pollingInterval = null;
    }
}