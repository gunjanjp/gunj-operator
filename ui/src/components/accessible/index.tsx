// Accessible React Components Library
// Gunj Operator UI Components with Built-in Accessibility

import React, { useId, useRef, useEffect, useState } from 'react';
import './accessible-components.css';

// ============================================
// Button Component with Accessibility
// ============================================
export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'danger' | 'ghost';
  size?: 'small' | 'medium' | 'large';
  loading?: boolean;
  icon?: React.ReactNode;
  iconPosition?: 'left' | 'right';
}

export const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ 
    children, 
    variant = 'primary', 
    size = 'medium', 
    loading = false,
    disabled = false,
    icon,
    iconPosition = 'left',
    className = '',
    onClick,
    ...props 
  }, ref) => {
    const handleClick = (e: React.MouseEvent<HTMLButtonElement>) => {
      if (!loading && !disabled && onClick) {
        onClick(e);
      }
    };

    return (
      <button
        ref={ref}
        className={`btn btn--${variant} btn--${size} ${className}`}
        disabled={disabled || loading}
        aria-busy={loading}
        aria-disabled={disabled || loading}
        onClick={handleClick}
        {...props}
      >
        {loading && (
          <span className="btn__spinner" aria-label="Loading">
            <Spinner />
          </span>
        )}
        {icon && iconPosition === 'left' && (
          <span className="btn__icon btn__icon--left" aria-hidden="true">
            {icon}
          </span>
        )}
        <span className="btn__content">{children}</span>
        {icon && iconPosition === 'right' && (
          <span className="btn__icon btn__icon--right" aria-hidden="true">
            {icon}
          </span>
        )}
      </button>
    );
  }
);

Button.displayName = 'Button';

// ============================================
// Form Field Component with Accessibility
// ============================================
export interface FormFieldProps {
  label: string;
  error?: string;
  hint?: string;
  required?: boolean;
  children: React.ReactElement;
}

export const FormField: React.FC<FormFieldProps> = ({
  label,
  error,
  hint,
  required = false,
  children
}) => {
  const fieldId = useId();
  const errorId = `${fieldId}-error`;
  const hintId = `${fieldId}-hint`;

  const ariaDescribedBy = [
    error && errorId,
    hint && hintId
  ].filter(Boolean).join(' ');

  return (
    <div className={`form-field ${error ? 'form-field--error' : ''}`}>
      <label htmlFor={fieldId} className="form-field__label">
        {label}
        {required && (
          <span className="form-field__required" aria-label="required">
            *
          </span>
        )}
      </label>
      
      {hint && (
        <div id={hintId} className="form-field__hint">
          {hint}
        </div>
      )}
      
      {React.cloneElement(children, {
        id: fieldId,
        'aria-invalid': !!error,
        'aria-describedby': ariaDescribedBy || undefined,
        'aria-required': required,
        className: `form-field__input ${children.props.className || ''}`
      })}
      
      {error && (
        <div id={errorId} role="alert" className="form-field__error">
          <ErrorIcon aria-hidden="true" />
          {error}
        </div>
      )}
    </div>
  );
};

// ============================================
// Modal/Dialog Component with Accessibility
// ============================================
export interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  children: React.ReactNode;
  size?: 'small' | 'medium' | 'large';
  closeOnOverlayClick?: boolean;
}

export const Modal: React.FC<ModalProps> = ({
  isOpen,
  onClose,
  title,
  children,
  size = 'medium',
  closeOnOverlayClick = true
}) => {
  const titleId = useId();
  const modalRef = useRef<HTMLDivElement>(null);
  const previousActiveElement = useRef<HTMLElement | null>(null);

  // Focus management
  useEffect(() => {
    if (isOpen) {
      previousActiveElement.current = document.activeElement as HTMLElement;
      
      // Focus the modal
      const focusableElements = modalRef.current?.querySelectorAll(
        'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
      );
      
      if (focusableElements && focusableElements.length > 0) {
        (focusableElements[0] as HTMLElement).focus();
      }
      
      // Prevent body scroll
      document.body.style.overflow = 'hidden';
      
      // Announce to screen readers
      announce(`${title} dialog opened`);
    } else {
      // Restore body scroll
      document.body.style.overflow = '';
      
      // Restore focus
      if (previousActiveElement.current) {
        previousActiveElement.current.focus();
      }
    }
    
    return () => {
      document.body.style.overflow = '';
    };
  }, [isOpen, title]);

  // Keyboard handling
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (!isOpen) return;
      
      if (e.key === 'Escape') {
        onClose();
      }
      
      // Focus trap
      if (e.key === 'Tab') {
        const focusableElements = modalRef.current?.querySelectorAll(
          'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
        );
        
        if (focusableElements && focusableElements.length > 0) {
          const firstElement = focusableElements[0] as HTMLElement;
          const lastElement = focusableElements[focusableElements.length - 1] as HTMLElement;
          
          if (e.shiftKey && document.activeElement === firstElement) {
            e.preventDefault();
            lastElement.focus();
          } else if (!e.shiftKey && document.activeElement === lastElement) {
            e.preventDefault();
            firstElement.focus();
          }
        }
      }
    };
    
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  return (
    <div className="modal-overlay" onClick={closeOnOverlayClick ? onClose : undefined}>
      <div 
        ref={modalRef}
        className={`modal modal--${size}`}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        onClick={(e) => e.stopPropagation()}
      >
        <header className="modal__header">
          <h2 id={titleId} className="modal__title">
            {title}
          </h2>
          <button
            className="modal__close"
            onClick={onClose}
            aria-label="Close dialog"
          >
            <CloseIcon aria-hidden="true" />
          </button>
        </header>
        
        <div className="modal__content">
          {children}
        </div>
      </div>
    </div>
  );
};

// ============================================
// Alert Component with Accessibility
// ============================================
export interface AlertProps {
  type: 'info' | 'success' | 'warning' | 'error';
  title?: string;
  children: React.ReactNode;
  onClose?: () => void;
  autoClose?: number;
}

export const Alert: React.FC<AlertProps> = ({
  type,
  title,
  children,
  onClose,
  autoClose
}) => {
  const alertRef = useRef<HTMLDivElement>(null);
  
  useEffect(() => {
    if (autoClose && onClose) {
      const timer = setTimeout(onClose, autoClose);
      return () => clearTimeout(timer);
    }
  }, [autoClose, onClose]);
  
  useEffect(() => {
    // Announce to screen readers
    alertRef.current?.focus();
  }, []);

  const alertRole = type === 'error' || type === 'warning' ? 'alert' : 'status';
  const alertLive = type === 'error' || type === 'warning' ? 'assertive' : 'polite';

  return (
    <div
      ref={alertRef}
      className={`alert alert--${type}`}
      role={alertRole}
      aria-live={alertLive}
      aria-atomic="true"
      tabIndex={-1}
    >
      <div className="alert__icon" aria-hidden="true">
        {type === 'info' && <InfoIcon />}
        {type === 'success' && <SuccessIcon />}
        {type === 'warning' && <WarningIcon />}
        {type === 'error' && <ErrorIcon />}
      </div>
      
      <div className="alert__content">
        {title && <div className="alert__title">{title}</div>}
        <div className="alert__message">{children}</div>
      </div>
      
      {onClose && (
        <button
          className="alert__close"
          onClick={onClose}
          aria-label="Dismiss alert"
        >
          <CloseIcon aria-hidden="true" />
        </button>
      )}
    </div>
  );
};

// ============================================
// Skip Link Component
// ============================================
export const SkipLink: React.FC<{ href: string; children: React.ReactNode }> = ({
  href,
  children
}) => {
  return (
    <a href={href} className="skip-link">
      {children}
    </a>
  );
};

// ============================================
// Loading Spinner Component
// ============================================
export interface SpinnerProps {
  size?: 'small' | 'medium' | 'large';
  label?: string;
}

export const Spinner: React.FC<SpinnerProps> = ({ 
  size = 'medium', 
  label = 'Loading...' 
}) => {
  return (
    <div 
      className={`spinner spinner--${size}`}
      role="status"
      aria-label={label}
    >
      <span className="visually-hidden">{label}</span>
      <svg 
        className="spinner__svg" 
        viewBox="0 0 50 50"
        aria-hidden="true"
      >
        <circle
          className="spinner__circle"
          cx="25"
          cy="25"
          r="20"
          fill="none"
          strokeWidth="5"
        />
      </svg>
    </div>
  );
};

// ============================================
// Progress Bar Component
// ============================================
export interface ProgressBarProps {
  value: number;
  max?: number;
  label?: string;
  showValue?: boolean;
}

export const ProgressBar: React.FC<ProgressBarProps> = ({
  value,
  max = 100,
  label,
  showValue = true
}) => {
  const percentage = Math.min(100, Math.max(0, (value / max) * 100));
  
  return (
    <div className="progress-bar">
      {label && (
        <div className="progress-bar__label">
          {label}
        </div>
      )}
      <div 
        className="progress-bar__track"
        role="progressbar"
        aria-valuenow={value}
        aria-valuemin={0}
        aria-valuemax={max}
        aria-label={label || `Progress: ${value} of ${max}`}
      >
        <div 
          className="progress-bar__fill"
          style={{ width: `${percentage}%` }}
        />
      </div>
      {showValue && (
        <div className="progress-bar__value">
          {Math.round(percentage)}%
        </div>
      )}
    </div>
  );
};

// ============================================
// Tabs Component with Accessibility
// ============================================
export interface Tab {
  id: string;
  label: string;
  content: React.ReactNode;
  disabled?: boolean;
}

export interface TabsProps {
  tabs: Tab[];
  defaultTab?: string;
  onChange?: (tabId: string) => void;
}

export const Tabs: React.FC<TabsProps> = ({ 
  tabs, 
  defaultTab,
  onChange 
}) => {
  const [activeTab, setActiveTab] = useState(defaultTab || tabs[0]?.id);
  const tablistRef = useRef<HTMLDivElement>(null);

  const handleTabChange = (tabId: string) => {
    setActiveTab(tabId);
    onChange?.(tabId);
  };

  const handleKeyDown = (e: React.KeyboardEvent, currentIndex: number) => {
    const enabledTabs = tabs.filter(tab => !tab.disabled);
    const enabledIndexes = tabs
      .map((tab, index) => ({ tab, index }))
      .filter(({ tab }) => !tab.disabled)
      .map(({ index }) => index);
    
    let newIndex = currentIndex;
    
    switch (e.key) {
      case 'ArrowLeft':
        e.preventDefault();
        const prevIndex = enabledIndexes.reverse().find(i => i < currentIndex);
        if (prevIndex !== undefined) {
          newIndex = prevIndex;
        } else {
          newIndex = enabledIndexes[enabledIndexes.length - 1];
        }
        break;
        
      case 'ArrowRight':
        e.preventDefault();
        const nextIndex = enabledIndexes.find(i => i > currentIndex);
        if (nextIndex !== undefined) {
          newIndex = nextIndex;
        } else {
          newIndex = enabledIndexes[0];
        }
        break;
        
      case 'Home':
        e.preventDefault();
        newIndex = enabledIndexes[0];
        break;
        
      case 'End':
        e.preventDefault();
        newIndex = enabledIndexes[enabledIndexes.length - 1];
        break;
        
      default:
        return;
    }
    
    const newTab = tabs[newIndex];
    if (newTab && !newTab.disabled) {
      handleTabChange(newTab.id);
      const tabButton = tablistRef.current?.children[newIndex] as HTMLElement;
      tabButton?.focus();
    }
  };

  return (
    <div className="tabs">
      <div 
        ref={tablistRef}
        className="tabs__list"
        role="tablist"
      >
        {tabs.map((tab, index) => (
          <button
            key={tab.id}
            className={`tabs__tab ${activeTab === tab.id ? 'tabs__tab--active' : ''}`}
            role="tab"
            aria-selected={activeTab === tab.id}
            aria-controls={`tabpanel-${tab.id}`}
            id={`tab-${tab.id}`}
            tabIndex={activeTab === tab.id ? 0 : -1}
            disabled={tab.disabled}
            onClick={() => handleTabChange(tab.id)}
            onKeyDown={(e) => handleKeyDown(e, index)}
          >
            {tab.label}
          </button>
        ))}
      </div>
      
      {tabs.map(tab => (
        <div
          key={tab.id}
          className="tabs__panel"
          role="tabpanel"
          id={`tabpanel-${tab.id}`}
          aria-labelledby={`tab-${tab.id}`}
          hidden={activeTab !== tab.id}
          tabIndex={0}
        >
          {tab.content}
        </div>
      ))}
    </div>
  );
};

// ============================================
// Utility Functions
// ============================================

// Announce to screen readers
export const announce = (message: string, priority: 'polite' | 'assertive' = 'polite') => {
  const announcer = document.createElement('div');
  announcer.setAttribute('role', priority === 'assertive' ? 'alert' : 'status');
  announcer.setAttribute('aria-live', priority);
  announcer.className = 'visually-hidden';
  announcer.textContent = message;
  
  document.body.appendChild(announcer);
  
  setTimeout(() => {
    document.body.removeChild(announcer);
  }, 1000);
};

// Focus management hook
export const useFocusTrap = (isActive: boolean) => {
  const containerRef = useRef<HTMLDivElement>(null);
  
  useEffect(() => {
    if (!isActive || !containerRef.current) return;
    
    const container = containerRef.current;
    const focusableElements = container.querySelectorAll(
      'a[href], button, textarea, input, select, [tabindex]:not([tabindex="-1"])'
    );
    
    if (focusableElements.length === 0) return;
    
    const firstElement = focusableElements[0] as HTMLElement;
    const lastElement = focusableElements[focusableElements.length - 1] as HTMLElement;
    
    firstElement.focus();
    
    const handleTab = (e: KeyboardEvent) => {
      if (e.key !== 'Tab') return;
      
      if (e.shiftKey) {
        if (document.activeElement === firstElement) {
          e.preventDefault();
          lastElement.focus();
        }
      } else {
        if (document.activeElement === lastElement) {
          e.preventDefault();
          firstElement.focus();
        }
      }
    };
    
    container.addEventListener('keydown', handleTab);
    
    return () => {
      container.removeEventListener('keydown', handleTab);
    };
  }, [isActive]);
  
  return containerRef;
};

// ============================================
// Icon Components (Placeholder)
// ============================================
const ErrorIcon = () => <span>⚠️</span>;
const SuccessIcon = () => <span>✅</span>;
const InfoIcon = () => <span>ℹ️</span>;
const WarningIcon = () => <span>⚠️</span>;
const CloseIcon = () => <span>✕</span>;
