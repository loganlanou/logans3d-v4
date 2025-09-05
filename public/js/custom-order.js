let currentStep = 1;
const totalSteps = 5;
const formData = {};

function changeStep(n) {
    const steps = document.querySelectorAll('.step-content');
    const prevBtn = document.getElementById('prevBtn');
    const nextBtn = document.getElementById('nextBtn');
    const submitBtn = document.getElementById('submitBtn');
    
    // Hide current step
    steps[currentStep - 1].classList.add('hidden');
    
    // Update step
    currentStep += n;
    
    // Ensure step is within bounds
    if (currentStep < 1) currentStep = 1;
    if (currentStep > totalSteps) currentStep = totalSteps;
    
    // Show new step
    steps[currentStep - 1].classList.remove('hidden');
    
    // Update progress bar
    updateProgressBar(currentStep);
    
    // Update buttons
    if (currentStep === 1) {
        prevBtn.classList.add('hidden');
    } else {
        prevBtn.classList.remove('hidden');
    }
    
    if (currentStep === totalSteps) {
        nextBtn.classList.add('hidden');
        submitBtn.classList.remove('hidden');
        populateReviewStep();
    } else {
        nextBtn.classList.remove('hidden');
        submitBtn.classList.add('hidden');
    }
    
    // Scroll to top of form
    document.querySelector('.bg-gradient-to-br').scrollIntoView({ behavior: 'smooth', block: 'start' });
}

function updateProgressBar(step) {
    const progressSteps = document.querySelectorAll('.progress-step');
    const progressBar = document.querySelector('.progress-bar-fill');
    
    progressSteps.forEach((el, index) => {
        if (index < step) {
            el.classList.add('completed');
            el.classList.remove('active');
        } else if (index === step - 1) {
            el.classList.add('active');
            el.classList.remove('completed');
        } else {
            el.classList.remove('active', 'completed');
        }
    });
    
    // Update progress bar width
    const progressPercentage = ((step - 1) / (totalSteps - 1)) * 100;
    progressBar.style.width = progressPercentage + '%';
}

function selectProjectType(type) {
    formData.projectType = type;
    // Update UI to show selection
    document.querySelectorAll('.project-type-card').forEach(card => {
        card.classList.remove('ring-2', 'ring-emerald-500', 'bg-emerald-500/10', 'border-emerald-500/50');
        card.classList.add('border-slate-600/50', 'bg-slate-700/30');
    });
    event.currentTarget.classList.remove('border-slate-600/50', 'bg-slate-700/30');
    event.currentTarget.classList.add('ring-2', 'ring-emerald-500', 'bg-emerald-500/10', 'border-emerald-500/50');
}

function handleFileUpload(event) {
    const file = event.target.files[0];
    if (file) {
        formData.modelFile = file;
        document.getElementById('file-name').textContent = file.name;
        document.getElementById('file-info').classList.remove('hidden');
    }
}

function updateMaterial(material) {
    formData.material = material;
    updatePriceEstimate();
}

function updateSize(size) {
    formData.size = size;
    updatePriceEstimate();
}

function updatePriceEstimate() {
    // Simplified price calculation
    const basePrices = {
        'pla': 10,
        'abs': 15,
        'petg': 20,
        'tpu': 25
    };
    const sizeMultipliers = {
        'small': 1,
        'medium': 2,
        'large': 3,
        'xlarge': 5
    };
    
    if (formData.material && formData.size) {
        const basePrice = basePrices[formData.material] || 10;
        const multiplier = sizeMultipliers[formData.size] || 1;
        const estimatedPrice = basePrice * multiplier;
        
        document.getElementById('price-estimate').textContent = '$' + estimatedPrice + ' - $' + (estimatedPrice * 1.5);
    }
}

function populateReviewStep() {
    // Populate review section with form data
    document.getElementById('review-project-type').textContent = formData.projectType || 'Not selected';
    document.getElementById('review-material').textContent = formData.material || 'Not selected';
    document.getElementById('review-size').textContent = formData.size || 'Not selected';
    document.getElementById('review-name').textContent = formData.name || 'Not provided';
    document.getElementById('review-email').textContent = formData.email || 'Not provided';
}

// Form submission
document.addEventListener('DOMContentLoaded', function() {
    const form = document.getElementById('customOrderForm');
    if (form) {
        form.addEventListener('submit', async (e) => {
            e.preventDefault();
            
            // Collect remaining form data
            formData.name = document.getElementById('name')?.value;
            formData.email = document.getElementById('email')?.value;
            formData.phone = document.getElementById('phone')?.value;
            formData.description = document.getElementById('description')?.value;
            
            // Submit form
            try {
                const response = await fetch('/custom/quote', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify(formData)
                });
                
                if (response.ok) {
                    // Show success message
                    alert('Your custom order request has been submitted! We\'ll contact you within 24 hours with a detailed quote.');
                    // Reset form
                    form.reset();
                    currentStep = 1;
                    changeStep(0);
                } else {
                    alert('There was an error submitting your request. Please try again.');
                }
            } catch (error) {
                console.error('Error:', error);
                alert('There was an error submitting your request. Please try again.');
            }
        });
    }
});