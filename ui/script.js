// API Base URL - backend runs on port 8080
const API_BASE_URL = 'http://localhost:8080';

// Global state
let currentPage = 1;
let currentQuery = '';
let totalPages = 1;
let isLoading = false;
let recentPaths = JSON.parse(localStorage.getItem('recentPaths')) || [];

// DOM Elements
const folderPathInput = document.getElementById('folder-path');
const suggestionsBtn = document.getElementById('suggestions-btn');
const suggestionsDropdown = document.getElementById('suggestions-dropdown');
const recentPathsSection = document.getElementById('recent-paths-section');
const recentPathsContainer = document.getElementById('recent-paths-container');
const indexBtn = document.getElementById('index-btn');
const indexStatus = document.getElementById('index-status');

const searchQueryInput = document.getElementById('search-query');
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
    
    // Setup suggestion items
    setupSuggestionItems();
    updateRecentPaths();
    
    // Close dropdown when clicking outside
    document.addEventListener('click', function(e) {
        if (!suggestionsDropdown.contains(e.target) && !suggestionsBtn.contains(e.target)) {
            suggestionsDropdown.classList.remove('active');
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
    showStatus(indexStatus, 'Indexing files...', 'loading');
    
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
            showStatus(indexStatus, `Successfully indexed files from ${folderPath}`, 'success');
            addToRecentPaths(folderPath);
        } else {
            const data = await response.json();
            showStatus(indexStatus, `Error: ${data.errors ? data.errors.join(', ') : 'Failed to index files'}`, 'error');
        }
    } catch (error) {
        showStatus(indexStatus, `Error: ${error.message}`, 'error');
    } finally {
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