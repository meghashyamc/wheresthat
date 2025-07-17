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
    
    const pathDiv = document.createElement('div');
    pathDiv.className = 'result-path';
    pathDiv.textContent = result.path || 'Unknown path';
    
    const contentDiv = document.createElement('div');
    contentDiv.className = 'result-content';
    
    if (result.content) {
        contentDiv.innerHTML = highlightSearchTerms(result.content, currentQuery);
    } else {
        contentDiv.textContent = 'No content preview available';
    }
    
    resultDiv.appendChild(pathDiv);
    resultDiv.appendChild(contentDiv);
    
    return resultDiv;
}

// Highlight search terms in content
function highlightSearchTerms(content, query) {
    if (!query) return content;
    
    const escapedContent = content.replace(/[&<>"']/g, function(m) {
        return {
            '&': '&amp;',
            '<': '&lt;',
            '>': '&gt;',
            '"': '&quot;',
            "'": '&#39;'
        }[m];
    });
    
    const words = query.split(/\s+/).filter(word => word.length > 0);
    let highlightedContent = escapedContent;
    
    words.forEach(word => {
        const regex = new RegExp(`(${word.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')})`, 'gi');
        highlightedContent = highlightedContent.replace(regex, '<span class="result-highlight">$1</span>');
    });
    
    return highlightedContent;
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