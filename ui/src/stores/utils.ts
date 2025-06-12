// Store utilities and custom hooks

import { useEffect } from 'react';
import { shallow } from 'zustand/shallow';
import type { StoreApi, UseBoundStore } from 'zustand';

/**
 * Custom hook for subscribing to store changes with cleanup
 */
export function useStoreSubscription<T>(
  store: UseBoundStore<StoreApi<T>>,
  selector: (state: T) => any,
  effect: (value: any) => void | (() => void),
  deps: any[] = []
) {
  useEffect(() => {
    // Get initial value and run effect
    const value = selector(store.getState());
    const cleanup = effect(value);
    
    // Subscribe to changes
    const unsubscribe = store.subscribe(
      (state) => selector(state),
      (value) => {
        const cleanup = effect(value);
        if (typeof cleanup === 'function') {
          return cleanup;
        }
      },
      {
        equalityFn: shallow,
      }
    );
    
    return () => {
      unsubscribe();
      if (typeof cleanup === 'function') {
        cleanup();
      }
    };
  }, deps);
}

/**
 * Create a selector hook with shallow comparison
 */
export function createSelectorHook<T, R>(
  store: UseBoundStore<StoreApi<T>>,
  selector: (state: T) => R
) {
  return () => store(selector, shallow);
}

/**
 * Debounce store updates
 */
export function debounceStoreUpdate<T extends (...args: any[]) => void>(
  fn: T,
  delay: number
): T {
  let timeoutId: NodeJS.Timeout;
  
  return ((...args: Parameters<T>) => {
    clearTimeout(timeoutId);
    timeoutId = setTimeout(() => fn(...args), delay);
  }) as T;
}

/**
 * Create async action with loading state
 */
export function createAsyncAction<T extends (...args: any[]) => Promise<any>>(
  action: T,
  options: {
    onSuccess?: (result: any) => void;
    onError?: (error: any) => void;
    setLoading?: (loading: boolean) => void;
    setError?: (error: string | null) => void;
  } = {}
) {
  return async (...args: Parameters<T>) => {
    const { onSuccess, onError, setLoading, setError } = options;
    
    try {
      setLoading?.(true);
      setError?.(null);
      
      const result = await action(...args);
      
      onSuccess?.(result);
      return result;
    } catch (error: any) {
      const errorMessage = error.message || 'An error occurred';
      setError?.(errorMessage);
      onError?.(error);
      throw error;
    } finally {
      setLoading?.(false);
    }
  };
}

/**
 * Store devtools connector
 */
export function connectStoreToDevtools<T>(
  store: UseBoundStore<StoreApi<T>>,
  name: string
) {
  if (process.env.NODE_ENV === 'development' && typeof window !== 'undefined') {
    const devtools = (window as any).__REDUX_DEVTOOLS_EXTENSION__;
    
    if (devtools) {
      const connection = devtools.connect({ name });
      
      // Send initial state
      connection.init(store.getState());
      
      // Subscribe to store changes
      store.subscribe((state) => {
        connection.send('STATE_UPDATE', state);
      });
      
      // Subscribe to devtools actions
      connection.subscribe((message: any) => {
        if (message.type === 'DISPATCH' && message.state) {
          store.setState(JSON.parse(message.state));
        }
      });
    }
  }
}

/**
 * Create store persist middleware with encryption
 */
export async function encrypt(data: string): Promise<string> {
  // In production, use proper encryption
  // This is a placeholder implementation
  return btoa(data);
}

export async function decrypt(data: string): Promise<string> {
  // In production, use proper decryption
  // This is a placeholder implementation
  return atob(data);
}

/**
 * Store reset utility
 */
export function createStoreResetter<T>(initialState: T) {
  return (set: any) => () => {
    set(initialState, true);
  };
}

/**
 * Batch store updates
 */
export function batchUpdates<T>(updates: Array<(state: T) => void>) {
  return (set: any, get: any) => {
    set((state: T) => {
      updates.forEach(update => update(state));
    });
  };
}

/**
 * Create computed selector with memoization
 */
export function createComputedSelector<T, Args extends any[], R>(
  selector: (state: T, ...args: Args) => R
) {
  const cache = new Map<string, R>();
  
  return (state: T, ...args: Args): R => {
    const key = JSON.stringify(args);
    
    if (cache.has(key)) {
      return cache.get(key)!;
    }
    
    const result = selector(state, ...args);
    cache.set(key, result);
    
    // Limit cache size
    if (cache.size > 100) {
      const firstKey = cache.keys().next().value;
      cache.delete(firstKey);
    }
    
    return result;
  };
}

/**
 * Store middleware for logging actions
 */
export const logger = (config: any) => (set: any, get: any, api: any) =>
  config(
    (...args: any[]) => {
      console.group('[Store Update]');
      console.log('Previous State:', get());
      set(...args);
      console.log('Next State:', get());
      console.groupEnd();
    },
    get,
    api
  );

/**
 * Create a bound store hook with TypeScript support
 */
export function createBoundStore<T>(store: UseBoundStore<StoreApi<T>>) {
  return {
    use: store,
    getState: store.getState,
    setState: store.setState,
    subscribe: store.subscribe,
  };
}