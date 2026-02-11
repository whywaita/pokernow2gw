// === WASM State ===
let wasmInstance = null;
let wasmMemory = null;
let wasmReady = false;

// Text encoder/decoder for string conversion
const encoder = new TextEncoder();
const decoder = new TextDecoder();

// === WASM Memory Management ===

function allocateString(str) {
    const bytes = encoder.encode(str);
    const ptr = wasmInstance.exports.malloc(bytes.length);
    const mem = new Uint8Array(wasmMemory.buffer);
    mem.set(bytes, ptr);
    return { ptr, length: bytes.length };
}

function readString(ptr, length) {
    const mem = new Uint8Array(wasmMemory.buffer);
    const bytes = mem.slice(ptr, ptr + length);
    return decoder.decode(bytes);
}

// === WASM Initialization ===

function initWasm() {
    const go = new Go();
    WebAssembly.instantiateStreaming(fetch("pokernow2gw.wasm"), go.importObject).then(async (result) => {
        wasmInstance = result.instance;
        // In Go 1.24+, memory is exported as "mem" not "memory"
        wasmMemory = wasmInstance.exports.mem;

        // Start Go runtime in background
        // It will run forever because main() contains select{}
        go.run(result.instance);

        // Mark as ready
        wasmReady = true;
        console.log("WASM module loaded successfully (using go:wasmexport with Go runtime)");
    }).catch((err) => {
        const errorDetail = `Error: ${err.toString()}
Stack: ${err.stack || 'N/A'}
User Agent: ${navigator.userAgent}
Time: ${new Date().toISOString()}`;
        showError("Failed to load WASM module. Click below for details.", errorDetail);
        console.error(err);
    });
}

// === UI Helper Functions ===

function showError(message, detail = null) {
    document.getElementById('errorMessage').textContent = message;
    document.getElementById('error').style.display = 'block';
    document.getElementById('success').style.display = 'none';

    // Handle error detail
    const detailWrapper = document.getElementById('errorDetailWrapper');
    const detailContainer = document.getElementById('errorDetailContainer');
    const detailElement = document.getElementById('errorDetail');
    const toggleElement = document.getElementById('errorDetailToggle');

    if (detail) {
        detailElement.textContent = detail;
        detailWrapper.style.display = 'block';
        detailContainer.style.display = 'none';
        toggleElement.innerHTML = '▶ Show details for developers';
    } else {
        detailWrapper.style.display = 'none';
    }
}

function toggleErrorDetail(event) {
    event.preventDefault();
    const container = document.getElementById('errorDetailContainer');
    const toggle = document.getElementById('errorDetailToggle');

    if (container.style.display === 'none') {
        container.style.display = 'block';
        toggle.innerHTML = '▼ Hide details';
    } else {
        container.style.display = 'none';
        toggle.innerHTML = '▶ Show details for developers';
    }
}

function copyErrorDetail() {
    const detail = document.getElementById('errorDetail').textContent;
    navigator.clipboard.writeText(detail).then(() => {
        showCopySuccess();
    }).catch((err) => {
        // Fallback for older browsers
        const textarea = document.createElement('textarea');
        textarea.value = detail;
        document.body.appendChild(textarea);
        textarea.select();
        document.execCommand('copy');
        document.body.removeChild(textarea);
        showCopySuccess();
    });
}

function showCopySuccess() {
    const btn = document.querySelector('#errorDetailContainer button');
    const originalText = btn.innerHTML;
    btn.innerHTML = '✓ Copied!';
    btn.disabled = true;
    setTimeout(() => {
        btn.innerHTML = originalText;
        btn.disabled = false;
    }, 2000);
}

function showSuccess(message, skippedHandsInfo = null) {
    document.getElementById('successMessage').textContent = message;
    document.getElementById('success').style.display = 'block';
    document.getElementById('error').style.display = 'none';

    // Handle skipped hands detail
    const detailWrapper = document.getElementById('skippedDetailWrapper');
    const detailContainer = document.getElementById('skippedDetailContainer');
    const detailElement = document.getElementById('skippedDetail');
    const toggleElement = document.getElementById('skippedDetailToggle');

    if (skippedHandsInfo && skippedHandsInfo.length > 0) {
        const detailText = formatSkippedHandsInfo(skippedHandsInfo);
        detailElement.textContent = detailText;
        detailWrapper.style.display = 'block';
        detailContainer.style.display = 'none';
        toggleElement.innerHTML = '▶ Show skipped hands details';
    } else {
        detailWrapper.style.display = 'none';
    }
}

function formatSkippedHandsInfo(skippedHandsInfo) {
    const reasonLabels = {
        'incomplete_hand': 'Incomplete hand (not properly closed)',
        'too_many_players': 'Too many players (> 10)',
        'filtered_out': 'Filtered out by player count filter'
    };

    // Count by reason for summary
    const reasonCounts = {};
    skippedHandsInfo.forEach(info => {
        const reason = info.reason;
        reasonCounts[reason] = (reasonCounts[reason] || 0) + 1;
    });

    let text = `=== Skipped Hands Report ===
Total skipped: ${skippedHandsInfo.length} hands
Time: ${new Date().toISOString()}

=== Summary by Reason ===
`;
    for (const [reason, count] of Object.entries(reasonCounts)) {
        text += `${reasonLabels[reason] || reason}: ${count} hands\n`;
    }

    text += `
=== Details ===
`;
    skippedHandsInfo.forEach((info, index) => {
        text += `--- Hand #${index + 1} ---
Hand Number: #${info.hand_number}
Hand ID: ${info.hand_id}
Reason: ${reasonLabels[info.reason] || info.reason}
Detail: ${info.detail}`;
        if (info.player_count) {
            text += `
Player Count: ${info.player_count}`;
        }
        if (info.raw_input && info.raw_input.length > 0) {
            text += `

Raw Input (${info.raw_input.length} lines):
----------------------------------------
${info.raw_input.join('\n')}
----------------------------------------`;
        }
        text += '\n\n';
    });

    return text.trim();
}

function toggleSkippedDetail(event) {
    event.preventDefault();
    const container = document.getElementById('skippedDetailContainer');
    const toggle = document.getElementById('skippedDetailToggle');

    if (container.style.display === 'none') {
        container.style.display = 'block';
        toggle.innerHTML = '▼ Hide skipped hands details';
    } else {
        container.style.display = 'none';
        toggle.innerHTML = '▶ Show skipped hands details';
    }
}

function copySkippedDetail() {
    const detail = document.getElementById('skippedDetail').textContent;
    navigator.clipboard.writeText(detail).then(() => {
        showSkippedCopySuccess();
    }).catch((err) => {
        // Fallback for older browsers
        const textarea = document.createElement('textarea');
        textarea.value = detail;
        document.body.appendChild(textarea);
        textarea.select();
        document.execCommand('copy');
        document.body.removeChild(textarea);
        showSkippedCopySuccess();
    });
}

function showSkippedCopySuccess() {
    const btn = document.querySelector('#skippedDetailContainer button');
    const originalText = btn.innerHTML;
    btn.innerHTML = '✓ Copied!';
    btn.disabled = true;
    setTimeout(() => {
        btn.innerHTML = originalText;
        btn.disabled = false;
    }, 2000);
}

function hideMessages() {
    document.getElementById('error').style.display = 'none';
    document.getElementById('success').style.display = 'none';
}

function copyToClipboard() {
    const hhOutput = document.getElementById('hhOutput');
    hhOutput.select();
    document.execCommand('copy');
    showSuccess("Copied to clipboard!");
}

// === Shared Conversion Helpers ===

function extractDayPlayed(hhOutput) {
    const dateMatch = hhOutput.match(/(\d{4})\/(\d{2})\/(\d{2}) (\d{2}):(\d{2}):(\d{2})/);
    if (dateMatch) {
        return `${dateMatch[1]}-${dateMatch[2]}-${dateMatch[3]}_${dateMatch[4]}-${dateMatch[5]}-${dateMatch[6]}`;
    }
    // Fallback to current date/time if not found
    const now = new Date();
    return now.toISOString().slice(0, 10) + '_' + now.toISOString().slice(11, 19).replace(/:/g, '-');
}

function downloadFile(content, filename) {
    const blob = new Blob([content], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
}

function callWasmParseCSV(csvInput, heroName, filterFlags, gameType, rakePercent, rakeCapBB) {
    const csvData = allocateString(csvInput);
    const heroData = allocateString(heroName);

    let resultInfoPtr;
    if (rakePercent !== undefined && rakeCapBB !== undefined) {
        // Cash game mode: pass rake parameters
        resultInfoPtr = wasmInstance.exports.parseCSV(
            csvData.ptr, csvData.length,
            heroData.ptr, heroData.length,
            filterFlags,
            gameType,
            rakePercent,
            rakeCapBB
        );
    } else {
        // Tournament mode: pass 0 for rake parameters
        resultInfoPtr = wasmInstance.exports.parseCSV(
            csvData.ptr, csvData.length,
            heroData.ptr, heroData.length,
            filterFlags,
            gameType,
            0,  // rakePercent
            0   // rakeCapBB
        );
    }

    // Read result info from memory (5 uint32 values: ptr, len, skippedHands, skippedDetailPtr, skippedDetailLen)
    const view = new DataView(wasmMemory.buffer);

    const resultPtr = view.getUint32(resultInfoPtr, true);     // little-endian
    const resultLen = view.getUint32(resultInfoPtr + 4, true);
    const skippedHands = view.getUint32(resultInfoPtr + 8, true);
    const skippedDetailPtr = view.getUint32(resultInfoPtr + 12, true);
    const skippedDetailLen = view.getUint32(resultInfoPtr + 16, true);

    // Read the result string
    const resultText = readString(resultPtr, resultLen);

    // Read skipped hands detail JSON if available
    let skippedHandsInfo = null;
    if (skippedDetailPtr > 0 && skippedDetailLen > 0) {
        const skippedDetailJson = readString(skippedDetailPtr, skippedDetailLen);
        try {
            skippedHandsInfo = JSON.parse(skippedDetailJson);
        } catch (e) {
            console.error("Failed to parse skipped hands info:", e);
        }
    }

    return { resultText, skippedHands, skippedHandsInfo };
}

function handleConversionResult(resultText, skippedHands, skippedHandsInfo, errorDetail) {
    // Hide loading
    document.getElementById('loading').style.display = 'none';
    document.getElementById('convertBtn').disabled = false;

    // Check if this is an error (skippedHands = 0xFFFFFFFF indicates error)
    if (skippedHands === 0xFFFFFFFF) {
        showError("Conversion failed. Click below for details.", errorDetail);
        return false;
    }

    // Show result
    document.getElementById('hhOutput').value = resultText;
    document.getElementById('resultContainer').style.display = 'block';

    let message = "Conversion successful!";
    if (skippedHands > 0) {
        message += ` (${skippedHands} hands were skipped)`;
    }
    showSuccess(message, skippedHandsInfo);

    // Scroll to result
    document.getElementById('resultContainer').scrollIntoView({ behavior: 'smooth' });
    return true;
}

// === Initialization ===

// Set up Ctrl+Enter shortcut for conversion
document.addEventListener('DOMContentLoaded', function() {
    document.getElementById('csvInput').addEventListener('keydown', function(e) {
        if (e.ctrlKey && e.key === 'Enter') {
            convertCSV();
        }
    });
});

// Initialize WASM on load
initWasm();
