let currentStep = 1;
const totalSteps = 5;
let formData = {};
let draftId = null;
let draftLoaded = false;

// Show the reset confirmation modal
function confirmReset() {
    const modal = document.getElementById('resetModal');
    const content = document.getElementById('resetModalContent');
    if (modal && content) {
        modal.classList.remove('hidden');
        // Trigger animation
        requestAnimationFrame(() => {
            content.classList.remove('scale-95', 'opacity-0');
            content.classList.add('scale-100', 'opacity-100');
        });
        // Prevent body scroll
        document.body.style.overflow = 'hidden';
    }
}

// Close the reset confirmation modal
function closeResetModal() {
    const modal = document.getElementById('resetModal');
    const content = document.getElementById('resetModalContent');
    if (modal && content) {
        content.classList.remove('scale-100', 'opacity-100');
        content.classList.add('scale-95', 'opacity-0');
        setTimeout(() => {
            modal.classList.add('hidden');
            document.body.style.overflow = '';
        }, 200);
    }
}

// Called when user confirms reset in modal
function confirmResetAction() {
    closeResetModal();
    resetForm();
}

// Close modal on Escape key
document.addEventListener('keydown', function(e) {
    if (e.key === 'Escape') {
        const modal = document.getElementById('resetModal');
        if (modal && !modal.classList.contains('hidden')) {
            closeResetModal();
        }
    }
});

// Reset the form and clear the draft
async function resetForm() {
    // Clear draft on server first
    if (draftId) {
        try {
            await fetch('/api/custom/draft', {
                method: 'DELETE'
            });
        } catch (error) {
            console.error('Error deleting draft:', error);
        }
    }

    // Reload the page to get a completely fresh start
    // This ensures all state is properly reset and no draft is loaded
    window.location.reload();
}

// Load existing draft on page load
async function loadDraft() {
    try {
        // Check if there's a resume parameter in the URL (from recovery email)
        const urlParams = new URLSearchParams(window.location.search);
        const resumeId = urlParams.get('resume');

        let response;
        if (resumeId) {
            // Load draft by ID (from recovery email link)
            response = await fetch(`/api/custom/draft/${resumeId}`);
            if (response.status === 410) {
                // Draft already completed
                alert('This quote has already been submitted. Starting a new quote.');
                // Clear the resume parameter from URL
                window.history.replaceState({}, '', '/custom');
                response = await fetch('/api/custom/draft');
            } else if (!response.ok) {
                console.error('Failed to load draft by ID, falling back to session draft');
                response = await fetch('/api/custom/draft');
            }
        } else {
            response = await fetch('/api/custom/draft');
        }

        if (!response.ok) return;

        const data = await response.json();

        // Pre-fill user info from Clerk if logged in
        if (data.user) {
            const nameField = document.getElementById('name');
            const emailField = document.getElementById('email');
            if (nameField && !nameField.value) nameField.value = data.user.name || '';
            if (emailField && !emailField.value) emailField.value = data.user.email || '';
        }

        // Restore draft if exists
        if (data.draft && data.draft.id) {
            draftId = data.draft.id;
            restoreFormState(data.draft);

            // Go to the saved step
            const savedStep = data.draft.current_step || 1;
            if (savedStep > 1) {
                goToStep(savedStep);
            }
            draftLoaded = true;

            // Clear the resume parameter from URL after loading
            if (resumeId) {
                window.history.replaceState({}, '', '/custom');
            }
        }
    } catch (error) {
        console.error('Error loading draft:', error);
    }
}

// Restore form state from draft data
function restoreFormState(draft) {
    // Step 1 - Project Type
    if (draft.project_type) {
        formData.projectType = draft.project_type;
        // Visually select the project type card
        const card = document.querySelector(`[data-project-type="${draft.project_type}"]`);
        if (card) {
            selectProjectType(draft.project_type, card);
        }
    }

    // Step 2 - Name/Email (lead capture)
    if (draft.name) {
        formData.name = draft.name;
        const nameField = document.getElementById('name');
        if (nameField) nameField.value = draft.name;
    }
    if (draft.email) {
        formData.email = draft.email;
        const emailField = document.getElementById('email');
        if (emailField) emailField.value = draft.email;
    }

    // Step 4 - Material, Size, Color, Budget
    if (draft.material) {
        formData.material = draft.material;
        updateMaterial(draft.material);
    }
    if (draft.size) {
        formData.size = draft.size;
        updateSize(draft.size);
    }
    if (draft.color) {
        formData.color = draft.color;
        selectColor(draft.color);
    }
    if (draft.budget) {
        formData.budget = draft.budget;
    }

    // Step 4 - Timeline, Description
    if (draft.timeline) {
        formData.timeline = draft.timeline;
        const timelineRadio = document.querySelector(`input[name="timeline"][value="${draft.timeline}"]`);
        if (timelineRadio) timelineRadio.checked = true;
    }
    if (draft.description) {
        formData.description = draft.description;
        const descField = document.getElementById('description');
        if (descField) descField.value = draft.description;
    }

    // Restore checkbox options
    if (draft.finishing) {
        formData.finishing = true;
        const finishingCheckbox = document.getElementById('finishing');
        if (finishingCheckbox) finishingCheckbox.checked = true;
    }
    if (draft.painting) {
        formData.painting = true;
        const paintingCheckbox = document.getElementById('painting');
        if (paintingCheckbox) paintingCheckbox.checked = true;
    }
    if (draft.rush) {
        formData.rush = true;
        const rushCheckbox = document.getElementById('rush');
        if (rushCheckbox) rushCheckbox.checked = true;
    }
    if (draft.need_design) {
        formData.needDesign = true;
        const needDesignCheckbox = document.getElementById('need-design');
        if (needDesignCheckbox) needDesignCheckbox.checked = true;
    }
}

// Go to a specific step
function goToStep(targetStep) {
    const steps = document.querySelectorAll('.step-content');
    const prevBtn = document.getElementById('prevBtn');
    const nextBtn = document.getElementById('nextBtn');
    const submitBtn = document.getElementById('submitBtn');
    const resetBtn = document.getElementById('resetBtn');

    // Hide current step
    steps[currentStep - 1].classList.add('hidden');

    // Update step
    currentStep = targetStep;

    // Show new step
    steps[currentStep - 1].classList.remove('hidden');

    // Update progress bar
    updateProgressBar(currentStep);

    // Update buttons
    if (currentStep === 1) {
        prevBtn.classList.add('hidden');
        resetBtn.classList.add('hidden');
    } else {
        prevBtn.classList.remove('hidden');
        resetBtn.classList.remove('hidden');
    }

    if (currentStep === totalSteps) {
        nextBtn.classList.add('hidden');
        submitBtn.classList.remove('hidden');
        populateReviewStep();
    } else {
        nextBtn.classList.remove('hidden');
        submitBtn.classList.add('hidden');
    }

    // Hide/show hero header based on step
    const heroHeader = document.getElementById('hero-header');
    if (heroHeader) {
        if (currentStep === 1) {
            heroHeader.classList.remove('hidden');
        } else {
            heroHeader.classList.add('hidden');
        }
    }
}

// Save draft to server
async function saveDraft(step) {
    const draftData = {
        step: step,
        project_type: formData.projectType || '',
        name: formData.name || document.getElementById('name')?.value?.trim() || '',
        email: formData.email || document.getElementById('email')?.value?.trim() || '',
        material: formData.material || '',
        size: formData.size || '',
        color: formData.color || '',
        budget: formData.budget || '',
        timeline: formData.timeline || document.querySelector('input[name="timeline"]:checked')?.value || '',
        description: formData.description || document.getElementById('description')?.value?.trim() || '',
        // Include checkbox options
        finishing: formData.finishing || document.getElementById('finishing')?.checked || false,
        painting: formData.painting || document.getElementById('painting')?.checked || false,
        rush: formData.rush || document.getElementById('rush')?.checked || false,
        need_design: formData.needDesign || document.getElementById('need-design')?.checked || false
    };

    try {
        const response = await fetch('/api/custom/draft', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(draftData)
        });

        if (response.ok) {
            const result = await response.json();
            if (result.draft_id) {
                draftId = result.draft_id;
            }
        }
    } catch (error) {
        console.error('Error saving draft:', error);
    }
}

// Helper function to show inline error message
function showError(elementId, message) {
    // Remove any existing error for this element
    clearError(elementId);

    const element = document.getElementById(elementId);
    if (!element) {
        // For step-level errors (like project type), show at step level
        const stepContainer = document.getElementById(`step-${currentStep}`);
        if (stepContainer) {
            const errorDiv = document.createElement('div');
            errorDiv.id = `error-${elementId}`;
            errorDiv.className = 'error-message bg-red-500/20 border border-red-500/50 text-red-300 px-4 py-3 rounded-lg mb-4 flex items-center gap-2';
            errorDiv.innerHTML = `<svg class="w-5 h-5 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clip-rule="evenodd"></path></svg><span>${message}</span>`;
            stepContainer.insertBefore(errorDiv, stepContainer.firstChild.nextSibling);
        }
        return;
    }

    // Add error styling to the input
    element.classList.add('border-red-500', 'ring-2', 'ring-red-500/50');

    // Create error message element
    const errorDiv = document.createElement('div');
    errorDiv.id = `error-${elementId}`;
    errorDiv.className = 'error-message text-red-400 text-sm mt-1 flex items-center gap-1';
    errorDiv.innerHTML = `<svg class="w-4 h-4 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clip-rule="evenodd"></path></svg><span>${message}</span>`;

    // Insert after the element
    element.parentNode.insertBefore(errorDiv, element.nextSibling);

    // Focus the element
    element.focus();
}

// Helper function to clear error message
function clearError(elementId) {
    const errorDiv = document.getElementById(`error-${elementId}`);
    if (errorDiv) {
        errorDiv.remove();
    }

    const element = document.getElementById(elementId);
    if (element) {
        element.classList.remove('border-red-500', 'ring-2', 'ring-red-500/50');
    }
}

// Clear all errors in current step
function clearAllErrors() {
    document.querySelectorAll('.error-message').forEach(el => el.remove());
    document.querySelectorAll('.border-red-500').forEach(el => {
        el.classList.remove('border-red-500', 'ring-2', 'ring-red-500/50');
    });
}

function changeStep(n) {
    const steps = document.querySelectorAll('.step-content');
    const prevBtn = document.getElementById('prevBtn');
    const nextBtn = document.getElementById('nextBtn');
    const submitBtn = document.getElementById('submitBtn');
    const resetBtn = document.getElementById('resetBtn');

    // Clear previous errors
    clearAllErrors();

    // Validate current step before moving forward
    if (n > 0 && !validateCurrentStep()) {
        return;
    }

    // Collect form data from current step before leaving
    collectCurrentStepData();

    // Save draft when moving forward
    if (n > 0) {
        saveDraft(currentStep);

        // Track step progression with GA4
        if (typeof Analytics !== 'undefined') {
            Analytics.customOrderStep(currentStep + n, {
                project_type: formData.projectType || ''
            });
        }
    }

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
        resetBtn.classList.add('hidden');
    } else {
        prevBtn.classList.remove('hidden');
        resetBtn.classList.remove('hidden');
    }

    if (currentStep === totalSteps) {
        nextBtn.classList.add('hidden');
        submitBtn.classList.remove('hidden');
        populateReviewStep();
    } else {
        nextBtn.classList.remove('hidden');
        submitBtn.classList.add('hidden');
    }

    // Hide/show hero header based on step
    const heroHeader = document.getElementById('hero-header');
    if (heroHeader) {
        if (currentStep === 1) {
            heroHeader.classList.remove('hidden');
        } else {
            heroHeader.classList.add('hidden');
        }
    }

    // Scroll to top of progress bar for smooth transition
    const progressBar = document.querySelector('.progress-step')?.parentElement;
    if (progressBar) {
        progressBar.scrollIntoView({ behavior: 'smooth', block: 'start' });
    }
}

function validateCurrentStep() {
    let isValid = true;

    switch (currentStep) {
        case 1: // Project Type
            if (!formData.projectType) {
                showError('project-type', 'Please select a project type before continuing.');
                isValid = false;
            }
            break;
        case 2: // Lead Capture (name & email)
            const name = document.getElementById('name')?.value?.trim();
            const email = document.getElementById('email')?.value?.trim();

            if (!name) {
                showError('name', 'Please enter your name.');
                isValid = false;
            }
            if (!email) {
                showError('email', 'Please enter your email address.');
                isValid = false;
            } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
                showError('email', 'Please enter a valid email address.');
                isValid = false;
            }
            break;
        case 3: // Model Selection - no required fields
            break;
        case 4: // Customization (material & size required)
            if (!formData.material) {
                showError('material', 'Please select a material before continuing.');
                isValid = false;
            }
            if (!formData.size) {
                showError('size', 'Please select a size before continuing.');
                isValid = false;
            }
            break;
    }
    return isValid;
}

function collectCurrentStepData() {
    // Collect data from current step before leaving
    switch (currentStep) {
        case 2: // Lead Capture (name & email)
            formData.name = document.getElementById('name')?.value?.trim() || '';
            formData.email = document.getElementById('email')?.value?.trim() || '';
            break;
        case 3: // Model Selection
            formData.needDesign = document.getElementById('need-design')?.checked || false;
            break;
        case 4: // Customization (material, size, color, options, timeline, description)
            formData.finishing = document.getElementById('finishing')?.checked || false;
            formData.painting = document.getElementById('painting')?.checked || false;
            formData.rush = document.getElementById('rush')?.checked || false;
            formData.description = document.getElementById('description')?.value?.trim() || '';
            const timelineRadio = document.querySelector('input[name="timeline"]:checked');
            formData.timeline = timelineRadio?.value || 'standard';
            break;
        case 5: // Review (collect terms)
            formData.termsAccepted = document.getElementById('terms')?.checked || false;
            break;
    }
}

function collectAllFormData() {
    // Comprehensive collection of all form data before submission
    // Step 2 - Lead Capture
    formData.name = document.getElementById('name')?.value?.trim() || '';
    formData.email = document.getElementById('email')?.value?.trim() || '';

    // Step 3 - Model Selection
    formData.needDesign = document.getElementById('need-design')?.checked || false;

    // Step 4 - Customization (material, size, color already in formData from selection)
    formData.finishing = document.getElementById('finishing')?.checked || false;
    formData.painting = document.getElementById('painting')?.checked || false;
    formData.rush = document.getElementById('rush')?.checked || false;
    formData.description = document.getElementById('description')?.value?.trim() || '';
    const timelineRadio = document.querySelector('input[name="timeline"]:checked');
    formData.timeline = timelineRadio?.value || 'standard';

    // Step 5 - Terms
    formData.termsAccepted = document.getElementById('terms')?.checked || false;
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

function selectProjectType(type, element) {
    formData.projectType = type;

    // Track step 1 (project type selection) with GA4
    if (typeof Analytics !== 'undefined') {
        Analytics.customOrderStep(1, {
            project_type: type
        });
    }

    // Clear any project type error
    clearError('project-type');

    // Update UI to show selection - clear all cards first
    const allCards = document.querySelectorAll('.project-type-card');
    allCards.forEach(card => {
        card.style.borderColor = '';
        card.style.backgroundColor = '';
        card.style.boxShadow = '';
        card.style.transform = '';
    });

    // Use the passed element directly, or try to find by data attribute as fallback
    let selectedCard = element;
    if (!selectedCard) {
        selectedCard = document.querySelector(`[data-project-type="${type}"]`);
    }
    // If we clicked on a child element (like the emoji, h3, or p), traverse up to find the card
    if (selectedCard && !selectedCard.classList.contains('project-type-card')) {
        selectedCard = selectedCard.closest('.project-type-card');
    }

    if (selectedCard) {
        selectedCard.style.borderColor = '#10b981';
        selectedCard.style.backgroundColor = 'rgba(16, 185, 129, 0.15)';
        selectedCard.style.boxShadow = '0 0 0 3px rgba(16, 185, 129, 0.5)';
        selectedCard.style.transform = 'scale(1.02)';
    }
}

function handleFileUpload(event) {
    const file = event.target.files[0];
    if (!file) return;

    // Clear any previous file errors
    clearError('model-file');

    // Validate file size (50MB max)
    const maxSize = 50 * 1024 * 1024;
    if (file.size > maxSize) {
        showError('model-file', 'File size must be less than 50MB.');
        event.target.value = '';
        return;
    }

    // Validate file type
    const allowedExtensions = ['.stl', '.obj', '.3mf', '.step', '.stp'];
    const ext = '.' + file.name.split('.').pop().toLowerCase();
    if (!allowedExtensions.includes(ext)) {
        showError('model-file', 'Please upload a valid 3D model file (.stl, .obj, .3mf, .step, .stp).');
        event.target.value = '';
        return;
    }

    formData.modelFile = file;
    document.getElementById('file-name').textContent = file.name;
    document.getElementById('file-info').classList.remove('hidden');
}

function updateMaterial(material) {
    formData.material = material;

    // Clear any material error
    clearError('material');

    // Clear all material buttons - use inline styles for reliable dark theme
    document.querySelectorAll('.material-btn').forEach(btn => {
        btn.style.borderColor = '';
        btn.style.boxShadow = '';
        btn.style.backgroundColor = '';
    });
    // Highlight selected material
    const selectedBtn = document.querySelector(`[data-material="${material}"]`);
    if (selectedBtn) {
        selectedBtn.style.borderColor = '#3b82f6';
        selectedBtn.style.boxShadow = '0 0 0 3px rgba(59, 130, 246, 0.5)';
        selectedBtn.style.backgroundColor = 'rgba(59, 130, 246, 0.1)';
    }
    updatePriceEstimate();
}

function updateSize(size) {
    formData.size = size;

    // Clear any size error
    clearError('size');

    // Clear all size buttons - use inline styles for reliable dark theme
    document.querySelectorAll('.size-btn').forEach(btn => {
        btn.style.borderColor = '';
        btn.style.boxShadow = '';
        btn.style.backgroundColor = '';
    });
    // Highlight selected size
    const selectedBtn = document.querySelector(`[data-size="${size}"]`);
    if (selectedBtn) {
        selectedBtn.style.borderColor = '#3b82f6';
        selectedBtn.style.boxShadow = '0 0 0 3px rgba(59, 130, 246, 0.5)';
        selectedBtn.style.backgroundColor = 'rgba(59, 130, 246, 0.1)';
    }
    updatePriceEstimate();
}

function selectColor(color) {
    formData.color = color;
    // Clear all color buttons
    document.querySelectorAll('.color-btn').forEach(btn => {
        btn.classList.remove('ring-4', 'ring-offset-2', 'ring-blue-500');
    });
    // Highlight selected color
    const selectedBtn = document.querySelector(`[data-color="${color}"]`);
    if (selectedBtn) {
        selectedBtn.classList.add('ring-4', 'ring-offset-2', 'ring-blue-500');
    }
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
    // Collect all form data first
    collectAllFormData();

    // Map values to display names
    const projectTypeNames = {
        'figurine': 'Figurines & Miniatures',
        'prototype': 'Prototypes & Parts',
        'decorative': 'Decorative Items',
        'custom': 'Something Else'
    };

    const materialNames = {
        'pla': 'PLA',
        'abs': 'ABS',
        'petg': 'PETG',
        'tpu': 'TPU'
    };

    const sizeNames = {
        'small': 'Small (< 5cm)',
        'medium': 'Medium (5-10cm)',
        'large': 'Large (10-20cm)',
        'xlarge': 'X-Large (> 20cm)'
    };

    const colorNames = {
        'red': 'Red',
        'blue': 'Blue',
        'green': 'Green',
        'yellow': 'Yellow',
        'purple': 'Purple',
        'black': 'Black',
        'white': 'White',
        'orange': 'Orange'
    };

    // Populate basic fields
    document.getElementById('review-project-type').textContent = projectTypeNames[formData.projectType] || formData.projectType || 'Not selected';
    document.getElementById('review-material').textContent = materialNames[formData.material] || formData.material || 'Not selected';
    document.getElementById('review-size').textContent = sizeNames[formData.size] || formData.size || 'Not selected';
    document.getElementById('review-color').textContent = colorNames[formData.color] || formData.color || 'Not selected';
    document.getElementById('review-timeline').textContent = formData.timeline === 'rush' ? 'Rush (24-48 hours)' : 'Standard (3-5 days)';

    // Contact info
    document.getElementById('review-name').textContent = formData.name || 'Not provided';
    document.getElementById('review-email').textContent = formData.email || 'Not provided';

    // Phone (hide row if empty)
    const phoneRow = document.getElementById('review-phone-row');
    const phoneEl = document.getElementById('review-phone');
    if (formData.phone) {
        phoneEl.textContent = formData.phone;
        phoneRow.classList.remove('hidden');
    } else {
        phoneRow.classList.add('hidden');
    }

    // Build options string
    const options = [];
    if (formData.finishing) options.push('Professional Finishing');
    if (formData.painting) options.push('Hand Painting');
    if (formData.rush) options.push('Rush Order');
    if (formData.needDesign) options.push('Design Help');
    const optionsEl = document.getElementById('review-options');
    const optionsRow = document.getElementById('review-options-row');
    if (options.length > 0) {
        optionsEl.textContent = options.join(', ');
        optionsRow.classList.remove('hidden');
    } else {
        optionsEl.textContent = 'None';
    }

    // File name (show row if file uploaded)
    const fileRow = document.getElementById('review-file-row');
    const fileEl = document.getElementById('review-file');
    if (formData.modelFile) {
        fileEl.textContent = formData.modelFile.name;
        fileRow.classList.remove('hidden');
    } else {
        fileRow.classList.add('hidden');
    }

    // Reference images (show row if uploaded)
    const refImagesRow = document.getElementById('review-reference-images-row');
    const refImagesEl = document.getElementById('review-reference-images');
    const refImages = document.getElementById('reference-images');
    if (refImages && refImages.files && refImages.files.length > 0) {
        const count = refImages.files.length;
        refImagesEl.textContent = count + ' image' + (count > 1 ? 's' : '') + ' uploaded';
        refImagesRow.classList.remove('hidden');
    } else {
        refImagesRow.classList.add('hidden');
    }

    // Description (show section if provided)
    const descSection = document.getElementById('review-description-section');
    const descEl = document.getElementById('review-description');
    if (formData.description) {
        descEl.textContent = formData.description;
        descSection.classList.remove('hidden');
    } else {
        descSection.classList.add('hidden');
    }
}

// Show success message inline
function showSuccessMessage(message) {
    const form = document.getElementById('customOrderForm');
    const successDiv = document.createElement('div');
    successDiv.id = 'success-message';
    successDiv.className = 'bg-emerald-500/20 border border-emerald-500/50 text-emerald-300 px-6 py-4 rounded-xl mb-6 flex items-center gap-3';
    successDiv.innerHTML = `<svg class="w-6 h-6 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd"></path></svg><span>${message}</span>`;
    form.parentNode.insertBefore(successDiv, form);
    form.classList.add('hidden');
}

// Show form error message inline
function showFormError(message) {
    clearError('form-error');
    const submitBtn = document.getElementById('submitBtn');
    const errorDiv = document.createElement('div');
    errorDiv.id = 'error-form-error';
    errorDiv.className = 'error-message bg-red-500/20 border border-red-500/50 text-red-300 px-4 py-3 rounded-lg mb-4 flex items-center gap-2';
    errorDiv.innerHTML = `<svg class="w-5 h-5 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clip-rule="evenodd"></path></svg><span>${message}</span>`;
    submitBtn.parentNode.insertBefore(errorDiv, submitBtn.parentNode.firstChild);
}

// Form submission
document.addEventListener('DOMContentLoaded', function() {
    // Load draft on page load
    loadDraft();

    const form = document.getElementById('customOrderForm');
    if (form) {
        // Clear errors on input change
        form.addEventListener('input', function(e) {
            if (e.target.id) {
                clearError(e.target.id);
            }
        });

        form.addEventListener('submit', async (e) => {
            e.preventDefault();

            // Clear previous errors
            clearAllErrors();

            // Collect all form data
            collectAllFormData();

            // Validate terms acceptance
            if (!formData.termsAccepted) {
                showError('terms', 'Please accept the Terms of Service to continue.');
                return;
            }

            // Build FormData for multipart submission (supports file uploads)
            const submitData = new FormData();

            // Append model file if provided
            if (formData.modelFile) {
                submitData.append('modelFile', formData.modelFile);
            }

            // Append reference images if provided
            const refImages = document.getElementById('reference-images');
            if (refImages && refImages.files.length > 0) {
                for (let i = 0; i < refImages.files.length; i++) {
                    submitData.append('referenceImages', refImages.files[i]);
                }
            }

            // Build JSON data object (excluding File objects)
            const jsonData = {
                projectType: formData.projectType || '',
                material: formData.material || '',
                size: formData.size || '',
                color: formData.color || '',
                name: formData.name || '',
                email: formData.email || '',
                phone: formData.phone || '',
                description: formData.description || '',
                timeline: formData.timeline || 'standard',
                finishing: formData.finishing || false,
                painting: formData.painting || false,
                rush: formData.rush || false,
                needDesign: formData.needDesign || false
            };
            submitData.append('data', JSON.stringify(jsonData));

            // Disable submit button while processing
            const submitBtn = document.getElementById('submitBtn');
            const originalText = submitBtn.textContent;
            submitBtn.disabled = true;
            submitBtn.textContent = 'Verifying...';

            try {
                // Get reCAPTCHA token
                const container = document.querySelector('[data-recaptcha-key]');
                const siteKey = container ? container.dataset.recaptchaKey : '';
                if (siteKey && typeof grecaptcha !== 'undefined') {
                    try {
                        const token = await grecaptcha.execute(siteKey, {action: 'custom_quote'});
                        submitData.append('g-recaptcha-response', token);
                    } catch (recaptchaError) {
                        console.error('reCAPTCHA error:', recaptchaError);
                        showFormError('Security verification failed. Please refresh the page and try again.');
                        return;
                    }
                }

                submitBtn.textContent = 'Submitting...';

                const response = await fetch('/custom/quote', {
                    method: 'POST',
                    body: submitData // No Content-Type header - browser sets multipart boundary
                });

                if (response.ok) {
                    // Clear the draft so refreshing starts fresh
                    if (draftId) {
                        try {
                            await fetch('/api/custom/draft', { method: 'DELETE' });
                        } catch (err) {
                            console.error('Error clearing draft:', err);
                        }
                    }
                    // Track generate_lead event with GA4
                    if (typeof Analytics !== 'undefined') {
                        Analytics.generateLead('custom_order', 150, {
                            project_type: formData.projectType || 'unknown',
                            has_model_file: !!formData.modelFile,
                            timeline: formData.timeline || 'standard'
                        });
                    }
                    // Track Lead event with Meta Pixel
                    if (typeof fbq !== 'undefined') {
                        fbq('track', 'Lead', {
                            content_name: 'Custom Quote Request',
                            content_category: formData.projectType || 'Custom Order'
                        });
                    }
                    // Show success message
                    showSuccessMessage('Your custom order request has been submitted! We\'ll contact you within 24 hours with a detailed quote.');
                } else {
                    const errorData = await response.json().catch(() => ({}));
                    showFormError(errorData.error || 'There was an error submitting your request. Please try again.');
                }
            } catch (error) {
                console.error('Error:', error);
                showFormError('There was an error submitting your request. Please try again.');
            } finally {
                submitBtn.disabled = false;
                submitBtn.textContent = originalText;
            }
        });
    }
});
