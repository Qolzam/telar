// AI Engine Control Panel JavaScript
let currentTab = 'rag';

// Initialize the application
document.addEventListener('DOMContentLoaded', function() {
    logMessage('AI Engine Control Panel initialized');
    loadStatus();
});

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
        
        if (embedTech) embedTech.textContent = data.embedding_provider;
        if (completionTech) completionTech.textContent = data.completion_provider;
        if (genCompletionTech) genCompletionTech.textContent = data.completion_provider;
        
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

// Handle content generation (placeholder for future feature)
async function handleGenerate() {
    const topic = document.getElementById('community-topic').value.trim();
    const style = document.getElementById('style-preference').value.trim();
    
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
    logMessage(`Starting content generation for topic: "${topic}"`, 'info');
    
    try {
        const startTime = Date.now();
        
        // Step 2: Process
        completeFlowStep('gen-step-input');
        highlightFlowStep('gen-step-process');
        logMessage('Processing input parameters...', 'info');
        
        // Simulate processing delay for visual effect
        await new Promise(resolve => setTimeout(resolve, 300));
        
        // Step 3: Generate
        completeFlowStep('gen-step-process');
        highlightFlowStep('gen-step-generate');
        logMessage('Generating content with AI...', 'info');
        
        // Simulate generation delay for visual effect
        await new Promise(resolve => setTimeout(resolve, 800));
        
        // Step 4: Format
        completeFlowStep('gen-step-generate');
        highlightFlowStep('gen-step-format');
        logMessage('Formatting output...', 'info');
        
        // Simulate formatting delay for visual effect
        await new Promise(resolve => setTimeout(resolve, 200));
        
        // Step 5: Output
        completeFlowStep('gen-step-format');
        highlightFlowStep('gen-step-output');
        
        const endTime = Date.now();
        const responseTime = endTime - startTime;
        
        // Complete the generation flow
        completeFlowStep('gen-step-output');
        
        showStatus('generate-status', `✅ Content generated successfully! (${responseTime}ms)`, 'success');
        logMessage(`Content generation complete for topic: "${topic}" (${responseTime}ms)`);
        
        // Placeholder for future implementation
        document.getElementById('generated-starters').innerHTML = `
            <div class="placeholder success">
                <h4>🎉 Generated Conversation Starters</h4>
                <p><strong>Topic:</strong> ${topic}</p>
                <p><strong>Style:</strong> ${style}</p>
                <p><strong>Response Time:</strong> ${responseTime}ms</p>
                <hr>
                <p><em>Note: This is a demo of the flow diagram. The actual content generation feature will be implemented in Phase 4.</em></p>
                <p>This will generate conversation starters based on community topics using the configured completion provider.</p>
            </div>
        `;
        
    } catch (error) {
        // Mark current step as error
        const activeStep = document.querySelector('.flow-step.active');
        if (activeStep) {
            errorFlowStep(activeStep.id);
        }
        showStatus('generate-status', `❌ Generation failed: ${error.message}`, 'error');
        logMessage(`Content generation error: ${error.message}`, 'error');
    } finally {
        generateBtn.disabled = false;
        generateBtn.textContent = '🎨 Generate Ideas';
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
