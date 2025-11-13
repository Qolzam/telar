// AI Engine Control Panel JavaScript
let currentTab = 'rag';

// theme management
let isDarkTheme = false;

// model configuration
let modelConfig = {
    provider: 'Unknown',
    model: 'Unknown'
};

// Initialize the application
document.addEventListener('DOMContentLoaded', function() {
    logMessage('AI Engine Control Panel initialized');
    loadStatus();
    initializeTheme();
    loadConcurrentStatus(); 
    loadModelConfig(); 
});

// Theme toggle functionality
function toggleTheme() {
    isDarkTheme = !isDarkTheme;
    document.body.classList.toggle('dark-theme', isDarkTheme);
    
    const themeIcon = document.getElementById('theme-icon');
    themeIcon.textContent = isDarkTheme ? '☀️' : '🌙';
    
    localStorage.setItem('ai-engine-theme', isDarkTheme ? 'dark' : 'light');
    
    logMessage(`Switched to ${isDarkTheme ? 'dark' : 'light'} theme`);
}

// Load concurrent request status (Phase 4 feature)
async function loadConcurrentStatus() {
    try {
        const response = await fetch('/api/v1/concurrent-status');
        const data = await response.json();
        
        if (data.status === 'success') {
            document.getElementById('max-concurrent').textContent = data.data.max_concurrent;
            document.getElementById('active-requests').textContent = data.data.active_requests;
            document.getElementById('available-slots').textContent = data.data.available_slots;
            document.getElementById('can-accept-request').textContent = data.data.can_accept_request ? 'Yes' : 'No';
            
            const canAcceptElement = document.getElementById('can-accept-request');
            canAcceptElement.style.color = data.data.can_accept_request ? '#4CAF50' : '#f44336';
            
            logMessage(`Concurrent status updated: ${data.data.active_requests}/${data.data.max_concurrent} active requests`);
        } else {
            logMessage('Failed to load concurrent status', 'error');
        }
    } catch (error) {
        logMessage(`Error loading concurrent status: ${error.message}`, 'error');
    }
}

// Load model configuration (Phase 4 feature)
async function loadModelConfig() {
    try {
        const response = await fetch('/api/v1/model-config');
        const data = await response.json();
        
        if (data.status === 'success') {
            modelConfig = {
                provider: data.data.provider_display,
                model: data.data.model
            };
            
            logMessage(`Model configuration loaded: ${modelConfig.provider} (${modelConfig.model})`);
        } else {
            logMessage('Failed to load model configuration', 'error');
        }
    } catch (error) {
        logMessage(`Error loading model configuration: ${error.message}`, 'error');
    }
}

// Initialize theme from localStorage
function initializeTheme() {
    const savedTheme = localStorage.getItem('ai-engine-theme');
    if (savedTheme === 'dark') {
        isDarkTheme = true;
        document.body.classList.add('dark-theme');
        document.getElementById('theme-icon').textContent = '☀️';
    } else {
        isDarkTheme = false;
        document.body.classList.remove('dark-theme');
        document.getElementById('theme-icon').textContent = '🌙';
    }
}

// Tab switching logic
function showTab(tabName) {
    // Hide all tabs
    document.querySelectorAll('.tab-content').forEach(tab => {
        tab.classList.remove('active');
    });
    document.querySelectorAll('.tab-button').forEach(btn => {
        btn.classList.remove('active');
    });
    
    // Show selected tab
    document.getElementById(tabName + '-tab').classList.add('active');
    
    // Activate the corresponding button
    document.querySelector(`button[onclick="showTab('${tabName}')"]`).classList.add('active');
    
    currentTab = tabName;
    logMessage(`Switched to ${tabName} tab`);
}

// Load status information
async function loadStatus() {
    try {
        const response = await fetch('/status');
        const data = await response.json();
        
        document.getElementById('embedding-provider').textContent = data.embedding_provider;
        document.getElementById('completion-provider').textContent = data.completion_provider;
        document.getElementById('status-text').textContent = data.status;
        
        // Update flow diagram tech labels
        const embedTech = document.getElementById('embed-tech');
        const completionTech = document.getElementById('completion-tech');
        const genCompletionTech = document.getElementById('gen-completion-tech');
        const modCompletionTech = document.getElementById('mod-completion-tech');
        
        if (embedTech) embedTech.textContent = data.embedding_provider;
        if (completionTech) completionTech.textContent = data.completion_provider;
        if (genCompletionTech) genCompletionTech.textContent = data.completion_provider;
        if (modCompletionTech) modCompletionTech.textContent = data.completion_provider;
        
        if (data.status === 'healthy') {
            document.getElementById('status-indicator').style.color = '#4CAF50';
        } else {
            document.getElementById('status-indicator').style.color = '#f44336';
        }
        
        logMessage(`Status loaded: Embedding=${data.embedding_provider}, Completion=${data.completion_provider}`);
    } catch (error) {
        logMessage(`Failed to load status: ${error.message}`, 'error');
        document.getElementById('status-text').textContent = 'Error';
        document.getElementById('status-indicator').style.color = '#f44336';
    }
}

// Flow diagram control utilities
function resetFlowDiagram() {
    const steps = ['step-ingest', 'step-embed', 'step-store', 'step-query', 'step-retrieve', 'step-generate'];
    steps.forEach(stepId => {
        const element = document.getElementById(stepId);
        if (element) {
            element.className = 'flow-step idle';
        }
    });
}

function resetGenerationFlowDiagram() {
    const steps = ['gen-step-input', 'gen-step-process', 'gen-step-generate', 'gen-step-format', 'gen-step-output'];
    steps.forEach(stepId => {
        const element = document.getElementById(stepId);
        if (element) {
            element.className = 'flow-step idle';
        }
    });
}

function resetModerationFlowDiagram() {
    const steps = ['mod-step-input', 'mod-step-analyze', 'mod-step-score', 'mod-step-decision', 'mod-step-result'];
    steps.forEach(stepId => {
        const element = document.getElementById(stepId);
        if (element) {
            element.className = 'flow-step idle';
        }
    });
}

function setFlowStep(stepId, state) {
    const element = document.getElementById(stepId);
    if (element) {
        element.className = `flow-step ${state}`;
    }
}

function highlightFlowStep(stepId) {
    setFlowStep(stepId, 'active');
}

function completeFlowStep(stepId) {
    setFlowStep(stepId, 'complete');
}

function errorFlowStep(stepId) {
    setFlowStep(stepId, 'error');
}

// Quick load functions
async function loadReadme() {
    logMessage('Loading README.md from GitHub...', 'info');
    try {
        const response = await fetch('https://raw.githubusercontent.com/qolzam/telar/main/README.md');
        const text = await response.text();
        document.getElementById('document-text').value = text;
        document.getElementById('source-metadata').value = 'github/readme.md';
        logMessage('README.md loaded successfully!');
    } catch (error) {
        logMessage(`Error loading README.md: ${error.message}`, 'error');
    }
}

async function loadContributing() {
    logMessage('Loading CONTRIBUTING.md from GitHub...', 'info');
    try {
        const response = await fetch('https://raw.githubusercontent.com/qolzam/telar/main/CONTRIBUTING.md');
        const text = await response.text();
        document.getElementById('document-text').value = text;
        document.getElementById('source-metadata').value = 'github/contributing.md';
        logMessage('CONTRIBUTING.md loaded successfully!');
    } catch (error) {
        logMessage(`Error loading CONTRIBUTING.md: ${error.message}`, 'error');
    }
}

async function loadCodeOfConduct() {
    logMessage('Loading CODE_OF_CONDUCT.md from GitHub...', 'info');
    try {
        const response = await fetch('https://raw.githubusercontent.com/qolzam/telar/main/CODE_OF_CONDUCT.md');
        const text = await response.text();
        document.getElementById('document-text').value = text;
        document.getElementById('source-metadata').value = 'github/code_of_conduct.md';
        logMessage('CODE_OF_CONDUCT.md loaded successfully!');
    } catch (error) {
        logMessage(`Error loading CODE_OF_CONDUCT.md: ${error.message}`, 'error');
    }
}

function clearDocument() {
    document.getElementById('document-text').value = '';
    document.getElementById('source-metadata').value = 'demo/document';
    logMessage('Document content cleared.');
}

// Handle document ingestion
async function handleIngest() {
    const text = document.getElementById('document-text').value.trim();
    const source = document.getElementById('source-metadata').value.trim();
    
    if (!text) {
        showStatus('ingest-status', 'Please enter document text', 'error');
        return;
    }
    
    if (!source) {
        showStatus('ingest-status', 'Please enter source metadata', 'error');
        return;
    }
    
    // Reset and start flow diagram
    resetFlowDiagram();
    
    const ingestBtn = document.getElementById('ingest-btn');
    ingestBtn.disabled = true;
    ingestBtn.textContent = '📥 Ingesting...';
    
    // Step 1: Ingest
    highlightFlowStep('step-ingest');
    logMessage('Starting document ingestion...', 'info');
    
    try {
        const startTime = Date.now();
        
        // Step 2: Embed
        completeFlowStep('step-ingest');
        highlightFlowStep('step-embed');
        logMessage('Generating embeddings...', 'info');
        
        // Simulate embedding delay for visual effect
        await new Promise(resolve => setTimeout(resolve, 500));
        
        const response = await fetch('/api/v1/ingest', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                text: text,
                metadata: {
                    source: source,
                    type: 'document'
                }
            })
        });
        
        // Step 3: Store
        completeFlowStep('step-embed');
        highlightFlowStep('step-store');
        logMessage('Storing in Weaviate...', 'info');
        
        const endTime = Date.now();
        const responseTime = endTime - startTime;
        
        const data = await response.json();
        
        if (response.ok) {
            // Complete the ingestion flow
            completeFlowStep('step-store');
            showStatus('ingest-status', `✅ Document ingested successfully! (${responseTime}ms)`, 'success');
            logMessage(`Document ingested: ${data.id} (${responseTime}ms)`);
            document.getElementById('document-text').value = '';
        } else {
            // Mark current step as error
            const activeStep = document.querySelector('.flow-step.active');
            if (activeStep) {
                errorFlowStep(activeStep.id);
            }
            showStatus('ingest-status', `❌ Error: ${data.error || 'Unknown error'}`, 'error');
            logMessage(`Ingest failed: ${data.error}`, 'error');
        }
    } catch (error) {
        // Mark current step as error
        const activeStep = document.querySelector('.flow-step.active');
        if (activeStep) {
            errorFlowStep(activeStep.id);
        }
        showStatus('ingest-status', `❌ Network error: ${error.message}`, 'error');
        logMessage(`Ingest network error: ${error.message}`, 'error');
    } finally {
        ingestBtn.disabled = false;
        ingestBtn.textContent = '📥 Ingest Knowledge';
    }
}

// Handle knowledge query
async function handleQuery() {
    const question = document.getElementById('query-input').value.trim();
    
    if (!question) {
        showStatus('query-status', 'Please enter a question', 'error');
        return;
    }
    
    // Reset and start flow diagram
    resetFlowDiagram();
    
    const queryBtn = document.getElementById('query-btn');
    queryBtn.disabled = true;
    queryBtn.textContent = '🔍 Thinking...';
    
    // Step 1: Query
    highlightFlowStep('step-query');
    logMessage(`Querying: "${question}"`, 'info');
    
    try {
        const startTime = Date.now();
        
        // Step 2: Retrieve
        completeFlowStep('step-query');
        highlightFlowStep('step-retrieve');
        logMessage('Searching knowledge base...', 'info');
        
        const response = await fetch('/api/v1/query', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                question: question,
                limit: 5
            })
        });
        
        // Step 3: Generate
        completeFlowStep('step-retrieve');
        highlightFlowStep('step-generate');
        logMessage('Generating response...', 'info');
        
        const endTime = Date.now();
        const responseTime = endTime - startTime;
        
        const data = await response.json();
        
        if (response.ok) {
            // Complete the query flow
            completeFlowStep('step-generate');
            displayAnswer(data.answer);
            displaySources(data.sources || []);
            logMessage(`✅ Query complete! (${responseTime}ms)`);
        } else {
            // Mark current step as error
            const activeStep = document.querySelector('.flow-step.active');
            if (activeStep) {
                errorFlowStep(activeStep.id);
            }
            showStatus('query-status', `❌ Error: ${data.error || 'Unknown error'}`, 'error');
            logMessage(`Query failed: ${data.error}`, 'error');
        }
    } catch (error) {
        // Mark current step as error
        const activeStep = document.querySelector('.flow-step.active');
        if (activeStep) {
            errorFlowStep(activeStep.id);
        }
        showStatus('query-status', `❌ Network error: ${error.message}`, 'error');
        logMessage(`Query network error: ${error.message}`, 'error');
    } finally {
        queryBtn.disabled = false;
        queryBtn.textContent = '🔍 Ask Question';
    }
}

async function handleGenerate() {
    const topic = document.getElementById('community-topic').value.trim();
    const style = document.getElementById('style-preference').value.trim() || 'engaging';
    
    if (!topic) {
        showStatus('generate-status', 'Please enter a community topic', 'error');
        return;
    }
    
    resetGenerationFlowDiagram();
    
    const generateBtn = document.getElementById('generate-btn');
    generateBtn.disabled = true;
    generateBtn.textContent = '🎨 Generating...';
    
    // Step 1: Input
    highlightFlowStep('gen-step-input');
    logMessage(`Starting content generation for topic: "${topic}" with style: "${style}"`, 'info');
    
    try {
        const startTime = Date.now();
        
        // Step 2: Process
        completeFlowStep('gen-step-input');
        highlightFlowStep('gen-step-process');
        logMessage('Processing input parameters...', 'info');
        
        // Step 3: Generate 
        completeFlowStep('gen-step-process');
        highlightFlowStep('gen-step-generate');
        logMessage('Generating conversation starters with AI...', 'info');
        
        const response = await fetch('/api/v1/generate/conversation-starters', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                community_topic: topic,
                style: style.toLowerCase()
            })
        });
        
        // Step 4: Format
        completeFlowStep('gen-step-generate');
        highlightFlowStep('gen-step-format');
        logMessage('Formatting conversation starters...', 'info');
        
        const endTime = Date.now();
        const responseTime = endTime - startTime;
        
        const data = await response.json();
        
        if (response.ok) {
            // Step 5: Output
            completeFlowStep('gen-step-format');
            highlightFlowStep('gen-step-output');
            logMessage('Displaying generated content...', 'info');
            
            // Complete the generation flow
            completeFlowStep('gen-step-output');
            
            // The API returns the array directly, not wrapped in a 'starters' property
            const starters = Array.isArray(data) ? data : [];
            showStatus('generate-status', `✅ Generated ${starters.length} conversation starters! (${responseTime}ms)`, 'success');
            logMessage(`Content generation complete: ${starters.length} starters for "${topic}" (${responseTime}ms)`);
            
            // Display the actual generated conversation starters
            displayGeneratedStarters(starters, topic, style, responseTime);
            
        } else {
            // Mark current step as error
            const activeStep = document.querySelector('.flow-step.active');
            if (activeStep) {
                errorFlowStep(activeStep.id);
            }
            showStatus('generate-status', `❌ Error: ${data.error || 'Unknown error'}`, 'error');
            logMessage(`Content generation failed: ${data.error}`, 'error');
            
            // Show error in the starters container
            document.getElementById('generated-starters').innerHTML = `
                <div class="placeholder error">
                    <h4>❌ Generation Failed</h4>
                    <p><strong>Error:</strong> ${data.error || 'Unknown error occurred'}</p>
                    <p><strong>Details:</strong> ${data.details || 'No additional details available'}</p>
                </div>
            `;
        }
        
    } catch (error) {
        // Mark current step as error
        const activeStep = document.querySelector('.flow-step.active');
        if (activeStep) {
            errorFlowStep(activeStep.id);
        }
        showStatus('generate-status', `❌ Network error: ${error.message}`, 'error');
        logMessage(`Content generation network error: ${error.message}`, 'error');
        
        // Show network error
        document.getElementById('generated-starters').innerHTML = `
            <div class="placeholder error">
                <h4>❌ Network Error</h4>
                <p><strong>Error:</strong> ${error.message}</p>
                <p>Please check your network connection and try again.</p>
            </div>
        `;
    } finally {
        generateBtn.disabled = false;
        generateBtn.textContent = '🎨 Generate Ideas';
    }
}

// Display generated conversation starters
function displayGeneratedStarters(starters, topic, style, responseTime) {
    const startersContainer = document.getElementById('generated-starters');
    
    let html = `
        <div class="starters-header">
            <h4>✨ Generated Conversation Starters</h4>
            <div class="generation-metadata">
                <span><strong>Topic:</strong> ${topic}</span>
                <span><strong>Style:</strong> ${style}</span>
                <span><strong>Count:</strong> ${starters.length}</span>
                <span><strong>Response Time:</strong> ${responseTime}ms</span>
                <span><strong>Model:</strong> ${modelConfig.provider} (${modelConfig.model})</span>
            </div>
        </div>
        <div class="starters-list">
    `;
    
    starters.forEach((starter, index) => {
        const starterId = `starter-${index}`;
        html += `
            <div class="starter-item" data-starter-id="${index}">
                <div class="starter-number">${index + 1}</div>
                <div class="starter-content" id="${starterId}">${starter}</div>
                <div class="starter-actions">
                    <button onclick="copyStarter('${starterId}')" class="copy-btn" title="Copy to clipboard">
                        📋 Copy
                    </button>
                    <button onclick="shareStarter('${starter.replace(/'/g, "\\'")}', '${topic}')" class="share-btn" title="Share starter">
                        🔗 Share
                    </button>
                </div>
            </div>
        `;
    });
    
    html += `
        </div>
        <div class="starters-footer">
            <button onclick="copyAllStarters()" class="copy-all-btn">📋 Copy All Starters</button>
            <button onclick="regenerateStarters()" class="regenerate-btn">🔄 Generate More</button>
        </div>
    `;
    
    startersContainer.innerHTML = html;
}

// Copy individual starter to clipboard
async function copyStarter(starterId) {
    const element = document.getElementById(starterId);
    const text = element.textContent;
    
    try {
        await navigator.clipboard.writeText(text);
        logMessage(`Copied starter: "${text.substring(0, 50)}..."`, 'info');
        
        // Visual feedback
        const copyBtn = element.parentElement.querySelector('.copy-btn');
        const originalText = copyBtn.textContent;
        copyBtn.textContent = '✅ Copied!';
        copyBtn.style.background = '#4CAF50';
        
        setTimeout(() => {
            copyBtn.textContent = originalText;
            copyBtn.style.background = '';
        }, 2000);
        
    } catch (error) {
        logMessage(`Failed to copy starter: ${error.message}`, 'error');
        // Fallback for older browsers
        element.select();
        document.execCommand('copy');
    }
}

// Share starter (opens share dialog or copies link)
function shareStarter(starter, topic) {
    const shareText = `Community conversation starter for "${topic}":\n\n${starter}\n\n#CommunityEngagement #${topic.replace(/\s+/g, '')}`;
    
    if (navigator.share) {
        // Use native share API if available
        navigator.share({
            title: `Conversation Starter: ${topic}`,
            text: shareText
        }).then(() => {
            logMessage('Starter shared successfully', 'info');
        }).catch((error) => {
            logMessage(`Share failed: ${error.message}`, 'error');
        });
    } else {
        // Fallback: copy to clipboard
        navigator.clipboard.writeText(shareText).then(() => {
            logMessage('Share text copied to clipboard', 'info');
            alert('Share text copied to clipboard!');
        }).catch((error) => {
            logMessage(`Failed to copy share text: ${error.message}`, 'error');
        });
    }
}

// Copy all starters to clipboard
async function copyAllStarters() {
    const starterElements = document.querySelectorAll('.starter-content');
    const starters = Array.from(starterElements).map((el, index) => `${index + 1}. ${el.textContent}`);
    const allText = starters.join('\n\n');
    
    try {
        await navigator.clipboard.writeText(allText);
        logMessage(`Copied all ${starters.length} conversation starters`, 'info');
        
        // Visual feedback
        const copyAllBtn = document.querySelector('.copy-all-btn');
        const originalText = copyAllBtn.textContent;
        copyAllBtn.textContent = '✅ All Copied!';
        copyAllBtn.style.background = '#4CAF50';
        
        setTimeout(() => {
            copyAllBtn.textContent = originalText;
            copyAllBtn.style.background = '';
        }, 2000);
        
    } catch (error) {
        logMessage(`Failed to copy all starters: ${error.message}`, 'error');
    }
}

// Regenerate starters with same parameters
function regenerateStarters() {
    const topic = document.getElementById('community-topic').value.trim();
    const style = document.getElementById('style-preference').value.trim();
    
    if (topic) {
        logMessage(`Regenerating starters for: "${topic}"`, 'info');
        handleGenerate();
    } else {
        showStatus('generate-status', 'Please enter a topic to regenerate starters', 'error');
    }
}

// Display answer with markdown support
function displayAnswer(answer) {
    const answerDiv = document.getElementById('answer-display');
    if (answer) {
        // Use marked.js to render markdown
        answerDiv.innerHTML = marked.parse(answer);
    } else {
        answerDiv.innerHTML = '<p>No answer generated.</p>';
    }
}

// Display sources
function displaySources(sources) {
    const sourcesDiv = document.getElementById('sources-display');
    const sourcesSection = document.getElementById('sources-section');
    
    if (!sources || sources.length === 0) {
        sourcesDiv.innerHTML = '<p>No sources found.</p>';
        sourcesSection.style.display = 'none';
        return;
    }
    
    sourcesSection.style.display = 'block';
    
    let html = '';
    sources.forEach((source, index) => {
        html += `
            <div class="source-item">
                <div class="source-score">Source #${index + 1} - Relevance Score: ${source.score.toFixed(3)}</div>
                <div class="source-text">${source.text.substring(0, 300)}${source.text.length > 300 ? '...' : ''}</div>
                ${source.metadata ? `<div class="source-metadata">Metadata: ${JSON.stringify(source.metadata)}</div>` : ''}
            </div>
        `;
    });
    
    sourcesDiv.innerHTML = html;
}

// Show status message
function showStatus(elementId, message, type = 'info') {
    const element = document.getElementById(elementId);
    if (element) {
        element.textContent = message;
        element.className = `status-message ${type}`;
        
        // Auto-hide after 5 seconds for success messages
        if (type === 'success') {
            setTimeout(() => {
                element.textContent = '';
                element.className = 'status-message';
            }, 5000);
        }
    }
}

// Log message to status log
function logMessage(message, type = 'info') {
    const logContainer = document.getElementById('log-container');
    const timestamp = new Date().toLocaleTimeString();
    const logEntry = document.createElement('div');
    logEntry.className = `log-entry ${type}`;
    logEntry.innerHTML = `<span class="log-time">[${timestamp}]</span> <span class="log-message">${message}</span>`;
    
    logContainer.appendChild(logEntry);
    logContainer.scrollTop = logContainer.scrollHeight;
    
    // Keep only last 50 log entries
    while (logContainer.children.length > 50) {
        logContainer.removeChild(logContainer.firstChild);
    }
}

// ============================================
// CONTENT MODERATION FUNCTIONS (Phase 5)
// ============================================

// Load example content for testing
function loadToxicExample() {
    document.getElementById('content-text').value = "You are a stupid idiot and I hate you! Go die!";
    logMessage('Loaded toxic content example', 'info');
}

function loadBenignExample() {
    document.getElementById('content-text').value = "I really enjoyed the new Go 1.23 features! The iterator support is amazing and makes code so much cleaner.";
    logMessage('Loaded benign content example', 'info');
}

function loadSpamExample() {
    document.getElementById('content-text').value = "BUY NOW!!! Click here for amazing deals!!! Limited time offer!!! www.spam-site.com GET RICH QUICK!!!";
    logMessage('Loaded spam content example', 'info');
}

function clearContent() {
    document.getElementById('content-text').value = '';
    logMessage('Content cleared', 'info');
}

// Handle content analysis
async function handleAnalyze() {
    const content = document.getElementById('content-text').value.trim();
    
    if (!content) {
        showStatus('analyze-status', 'Please enter content to analyze', 'error');
        return;
    }
    
    // Reset and start flow diagram
    resetModerationFlowDiagram();
    
    const analyzeBtn = document.getElementById('analyze-btn');
    analyzeBtn.disabled = true;
    analyzeBtn.textContent = '🔍 Analyzing...';
    
    // Step 1: Input
    highlightFlowStep('mod-step-input');
    logMessage(`Starting content moderation analysis (${content.length} chars)`, 'info');
    
    try {
        const startTime = Date.now();
        
        // Step 2: Analyze
        completeFlowStep('mod-step-input');
        highlightFlowStep('mod-step-analyze');
        logMessage('Sending content to AI analyzer...', 'info');
        
        const response = await fetch('/api/v1/analyze/content', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                content: content
            })
        });
        
        // Step 3: Score
        completeFlowStep('mod-step-analyze');
        highlightFlowStep('mod-step-score');
        logMessage('Processing AI analysis scores...', 'info');
        
        const endTime = Date.now();
        const responseTime = endTime - startTime;
        
        const data = await response.json();
        
        if (response.ok) {
            // Step 4: Decision
            completeFlowStep('mod-step-score');
            highlightFlowStep('mod-step-decision');
            logMessage('Making moderation decision...', 'info');
            
            // Step 5: Result
            completeFlowStep('mod-step-decision');
            highlightFlowStep('mod-step-result');
            logMessage('Displaying results...', 'info');
            
            // Complete the moderation flow
            completeFlowStep('mod-step-result');
            
            // Display results
            displayModerationResults(data, content, responseTime);
            
            const status = data.is_flagged ? 'FLAGGED' : 'APPROVED';
            showStatus('analyze-status', `✅ Analysis complete: ${status} (${responseTime}ms)`, data.is_flagged ? 'warning' : 'success');
            logMessage(`Content ${status}: ${data.is_flagged ? data.flag_reason : 'No violations detected'} (${responseTime}ms)`);
            
        } else {
            // Mark current step as error
            const activeStep = document.querySelector('.flow-step.active');
            if (activeStep) {
                errorFlowStep(activeStep.id);
            }
            showStatus('analyze-status', `❌ Error: ${data.error || 'Unknown error'}`, 'error');
            logMessage(`Analysis failed: ${data.error}`, 'error');
            
            // Show error in results
            document.getElementById('moderation-results').innerHTML = `
                <div class="placeholder error">
                    <h4>❌ Analysis Failed</h4>
                    <p><strong>Error:</strong> ${data.error || 'Unknown error occurred'}</p>
                    <p><strong>Details:</strong> ${data.details || 'No additional details available'}</p>
                </div>
            `;
        }
        
    } catch (error) {
        // Mark current step as error
        const activeStep = document.querySelector('.flow-step.active');
        if (activeStep) {
            errorFlowStep(activeStep.id);
        }
        showStatus('analyze-status', `❌ Network error: ${error.message}`, 'error');
        logMessage(`Analysis network error: ${error.message}`, 'error');
        
        // Show network error
        document.getElementById('moderation-results').innerHTML = `
            <div class="placeholder error">
                <h4>❌ Network Error</h4>
                <p><strong>Error:</strong> ${error.message}</p>
                <p>Please check your network connection and try again.</p>
            </div>
        `;
    } finally {
        analyzeBtn.disabled = false;
        analyzeBtn.textContent = '🔍 Analyze Content';
    }
}

// Display moderation results
function displayModerationResults(data, content, responseTime) {
    const resultsContainer = document.getElementById('moderation-results');
    const scoresContainer = document.getElementById('scores-display');
    
    // Main result display
    const statusIcon = data.is_flagged ? '🚫' : '✅';
    const statusText = data.is_flagged ? 'FLAGGED' : 'APPROVED';
    const statusClass = data.is_flagged ? 'flagged' : 'approved';
    
    let html = `
        <div class="moderation-result ${statusClass}">
            <div class="result-header">
                <div class="result-icon">${statusIcon}</div>
                <div class="result-status">
                    <h3>Content ${statusText}</h3>
                    <p class="result-confidence">Confidence: ${(data.confidence * 100).toFixed(0)}%</p>
                </div>
            </div>
    `;
    
    if (data.is_flagged && data.flag_reason) {
        html += `
            <div class="result-reason">
                <h4>🔍 Reason for Flagging:</h4>
                <p>${data.flag_reason}</p>
            </div>
        `;
    }
    
    html += `
            <div class="result-metadata">
                <span><strong>Analysis Time:</strong> ${responseTime}ms</span>
                <span><strong>Content Length:</strong> ${content.length} chars</span>
                <span><strong>Timestamp:</strong> ${new Date(data.timestamp).toLocaleString()}</span>
                <span><strong>Model:</strong> ${modelConfig.provider}</span>
            </div>
        </div>
    `;
    
    resultsContainer.innerHTML = html;
    
    // Scores display
    displayScores(data.scores, data.is_flagged);
}

// Display category scores with visual bars
function displayScores(scores, isFlagged) {
    const scoresContainer = document.getElementById('scores-display');
    
    const categories = [
        { key: 'toxicity', label: 'Toxicity', icon: '😡', description: 'Hate speech, harassment, threats' },
        { key: 'sexual', label: 'Sexual Content', icon: '🔞', description: 'Explicit or inappropriate sexual material' },
        { key: 'violence', label: 'Violence', icon: '⚔️', description: 'Graphic violence, gore, violent threats' },
        { key: 'spam', label: 'Spam', icon: '📧', description: 'Repetitive, promotional, low-quality content' },
        { key: 'misinformation', label: 'Misinformation', icon: '❌', description: 'False or misleading information' }
    ];
    
    let html = '<div class="scores-grid">';
    
    categories.forEach(category => {
        const score = scores[category.key] || 0;
        const percentage = (score * 100).toFixed(0);
        const isFlaggedCategory = score >= 0.7;
        const barClass = isFlaggedCategory ? 'danger' : (score >= 0.5 ? 'warning' : 'safe');
        
        html += `
            <div class="score-card ${isFlaggedCategory ? 'flagged' : ''}">
                <div class="score-header">
                    <span class="score-icon">${category.icon}</span>
                    <span class="score-label">${category.label}</span>
                    <span class="score-value">${percentage}%</span>
                </div>
                <div class="score-bar-container">
                    <div class="score-bar ${barClass}" style="width: ${percentage}%"></div>
                </div>
                <div class="score-description">${category.description}</div>
                ${isFlaggedCategory ? '<div class="score-alert">⚠️ Exceeds threshold (70%)</div>' : ''}
            </div>
        `;
    });
    
    html += '</div>';
    
    // Add threshold info
    html += `
        <div class="threshold-info">
            <p><strong>Flagging Threshold:</strong> Content is flagged if any category score exceeds 70% (0.7)</p>
            <p><strong>Current Status:</strong> ${isFlagged ? '🚫 Content flagged for review' : '✅ Content approved'}</p>
        </div>
    `;
    
    scoresContainer.innerHTML = html;
}
