// Payment flow state machine
class PaymentFlow {
    constructor() {
        this.currentStep = 1;
        this.details = {};
        this.initializeSSE();
        this.startDemoFlow();
    }

    initializeSSE() {
        // In production, connect to real /hub/sse endpoint
        // For now, we just show the flow with demo data
        console.log('Payment Flow Visualizer Ready');
    }

    updateStep(stepNumber, status, details = {}) {
        this.currentStep = stepNumber;
        this.details = { ...this.details, ...details };
        this.renderUI();
    }

    renderUI() {
        // Update timeline
        for (let i = 1; i <= 5; i++) {
            const el = document.getElementById(`timeline-${i}`);
            el.classList.remove('active', 'completed');
            if (i < this.currentStep) {
                el.classList.add('completed');
            } else if (i === this.currentStep) {
                el.classList.add('active');
            }
        }

        // Update step cards
        const steps = [
            { id: 'step-challenge', title: '1️⃣ Challenge Issued', message: 'Generating payment challenge...' },
            { id: 'step-sign', title: '2️⃣ Agent Signs', message: 'Signing with EVM_PRIVATE_KEY...' },
            { id: 'step-verify', title: '3️⃣ Hub Verifies', message: 'Verifying signature...' },
            { id: 'step-settle', title: '4️⃣ Settlement', message: 'Submitting to blockchain...' },
            { id: 'step-confirm', title: '5️⃣ Confirmed', message: 'Waiting for confirmation...' },
            { id: 'step-deliver', title: '6️⃣ Delivered', message: 'Returning results...' }
        ];

        steps.forEach((step, index) => {
            const el = document.getElementById(step.id);
            const stepNum = index + 1;

            el.classList.remove('active', 'completed');

            if (stepNum < this.currentStep) {
                el.classList.add('completed');
                el.querySelector('.status').innerHTML = `
                    <span class="status-indicator"></span>
                    ✓ Completed
                `;
            } else if (stepNum === this.currentStep) {
                el.classList.add('active');
                el.querySelector('.status').innerHTML = `
                    <span class="status-indicator"></span>
                    ${step.message}
                `;
            } else {
                el.querySelector('.status').textContent = 'Pending...';
            }
        });

        // Update details
        this.renderDetails();
        this.renderMessage();
    }

    renderDetails() {
        const container = document.getElementById('details-container');
        const lines = [];

        if (this.details.nonce) {
            lines.push(`<div class="detail-line"><span class="detail-label">Nonce:</span> <span class="detail-value">${this.details.nonce}</span></div>`);
        }
        if (this.details.price) {
            lines.push(`<div class="detail-line"><span class="detail-label">Price:</span> <span class="detail-value">${this.details.price} USDC</span></div>`);
        }
        if (this.details.payTo) {
            lines.push(`<div class="detail-line"><span class="detail-label">Pay To:</span> <span class="detail-value">${this.details.payTo}</span></div>`);
        }
        if (this.details.txHash) {
            lines.push(`<div class="detail-line"><span class="detail-label">Tx Hash:</span> <span class="detail-value">${this.details.txHash}</span></div>`);
        }
        if (this.details.agent) {
            lines.push(`<div class="detail-line"><span class="detail-label">Agent:</span> <span class="detail-value">${this.details.agent}</span></div>`);
        }

        if (lines.length === 0) {
            lines.push('<div class="detail-line"><span class="detail-label">Waiting for flow update...</span></div>');
        }

        container.innerHTML = lines.join('');
    }

    renderMessage() {
        const msg = document.getElementById('status-message');
        const messages = [
            { text: 'Generating payment challenge...', class: 'info' },
            { text: 'Agent signing payment proof...', class: 'info' },
            { text: 'Hub verifying signature via Coinbase x402...', class: 'info' },
            { text: 'Submitting to SettlementContract...', class: 'info' },
            { text: 'Waiting for blockchain confirmation...', class: 'info' },
            { text: '✓ Payment confirmed! Returning results...', class: 'success' }
        ];

        const current = messages[this.currentStep - 1] || messages[0];
        msg.textContent = current.text;
        msg.className = `status-message ${current.class}`;
    }

    startDemoFlow() {
        // Demo flow: automatically progress through steps
        const steps = [
            {
                step: 1,
                details: {
                    nonce: '0x' + 'a'.repeat(64),
                    price: '0.001',
                    payTo: '0x742d35Cc6634C0532925a3b844Bc9e7595f42521'
                }
            },
            {
                step: 2,
                details: {
                    agent: '0xc0ffee254729296a45a3885639ac7e10f9d54979'
                }
            },
            {
                step: 3,
                details: {}
            },
            {
                step: 4,
                details: {
                    txHash: '0x1234567890abcdef'
                }
            },
            {
                step: 5,
                details: {}
            },
            {
                step: 6,
                details: {}
            }
        ];

        let currentIndex = 0;
        const progressFlow = () => {
            if (currentIndex < steps.length) {
                const stage = steps[currentIndex];
                this.updateStep(stage.step, 'in-progress', stage.details);
                currentIndex++;
                setTimeout(progressFlow, 2000);
            }
        };

        // Start demo after 500ms
        setTimeout(progressFlow, 500);
    }
}

// Initialize on page load
document.addEventListener('DOMContentLoaded', () => {
    new PaymentFlow();
});
