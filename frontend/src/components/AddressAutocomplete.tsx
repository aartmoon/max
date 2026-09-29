import {
  useEffect,
  useId,
  useRef,
  useState,
  type KeyboardEvent,
} from "react";
import { addressApi } from "../api";
import type { AddressKind, AddressSuggestion } from "../types";

type Props = {
  label: string;
  kind: AddressKind;
  value: AddressSuggestion | null;
  onChange: (value: AddressSuggestion | null) => void;
  parentObjectId?: string;
  required?: boolean;
  placeholder?: string;
};

export function AddressAutocomplete({
  label,
  kind,
  value,
  onChange,
  parentObjectId,
  required = false,
  placeholder,
}: Props) {
  const inputId = useId();
  const listId = `${inputId}-list`;
  const rootRef = useRef<HTMLDivElement>(null);
  const itemLabel = (item: AddressSuggestion) =>
    kind === "apartment" ? item.displayName || item.fullAddress : item.fullAddress;
  const [query, setQuery] = useState(value ? itemLabel(value) : "");
  const [items, setItems] = useState<AddressSuggestion[]>([]);
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState("");
  const [activeIndex, setActiveIndex] = useState(-1);
  const [searchTick, setSearchTick] = useState(0);
  const canSearch = query.trim().length >= 2 || Boolean(parentObjectId);

  useEffect(() => {
    if (value) setQuery(itemLabel(value));
  }, [value, kind]);

  const close = () => {
    setItems([]);
    setMessage("");
    setActiveIndex(-1);
  };

  useEffect(() => {
    const onPointerDown = (event: PointerEvent) => {
      if (
        rootRef.current &&
        event.target instanceof Node &&
        !rootRef.current.contains(event.target)
      ) {
        close();
      }
    };
    document.addEventListener("pointerdown", onPointerDown);
    return () => document.removeEventListener("pointerdown", onPointerDown);
  }, []);

  useEffect(() => {
    if (value && query === itemLabel(value)) {
      close();
      return;
    }
    if (!canSearch) {
      close();
      return;
    }
    const controller = new AbortController();
    const timer = window.setTimeout(async () => {
      setLoading(true);
      setMessage("");
      try {
        const result = await addressApi.search({
          q: query.trim(),
          kind,
          parentObjectId,
          signal: controller.signal,
        });
        setItems(result.items);
        setActiveIndex(-1);
        if (result.items.length === 0) setMessage("Ничего не найдено");
      } catch (error) {
        if (error instanceof DOMException && error.name === "AbortError") return;
        setItems([]);
        setMessage((error as Error).message);
      } finally {
        if (!controller.signal.aborted) setLoading(false);
      }
    }, 300);
    return () => {
      window.clearTimeout(timer);
      controller.abort();
    };
  }, [canSearch, kind, parentObjectId, query, searchTick, value]);

  const select = (item: AddressSuggestion) => {
    setQuery(itemLabel(item));
    close();
    onChange(item);
  };

  const onKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (items.length === 0) return;
    if (event.key === "ArrowDown") {
      event.preventDefault();
      setActiveIndex((current) => (current + 1) % items.length);
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      setActiveIndex((current) => (current <= 0 ? items.length - 1 : current - 1));
    } else if (event.key === "Enter" && activeIndex >= 0) {
      event.preventDefault();
      select(items[activeIndex]);
    } else if (event.key === "Escape") {
      close();
    }
  };

  return (
    <div className="address-autocomplete" ref={rootRef}>
      <label htmlFor={inputId}>
        {label} {required && <span>*</span>}
      </label>
      <input
        id={inputId}
        role="combobox"
        aria-autocomplete="list"
        aria-expanded={items.length > 0}
        aria-controls={listId}
        aria-activedescendant={activeIndex >= 0 ? `${listId}-${activeIndex}` : undefined}
        required={required}
        value={query}
        placeholder={placeholder}
        autoComplete="off"
        onKeyDown={onKeyDown}
        onFocus={() => {
          if (canSearch && !(value && query === itemLabel(value))) {
            setSearchTick((current) => current + 1);
          }
        }}
        onBlur={() => {
          window.setTimeout(() => {
            if (!rootRef.current?.contains(document.activeElement)) close();
          }, 0);
        }}
        onChange={(event) => {
          const next = event.target.value;
          setQuery(next);
          if (value && next !== itemLabel(value)) onChange(null);
        }}
      />
      {loading && <small className="address-status">Ищем адрес…</small>}
      {!loading && message && <small className="address-status">{message}</small>}
      {items.length > 0 && (
        <div className="address-options" id={listId} role="listbox">
          {items.map((item, index) => (
            <button
              type="button"
              role="option"
              aria-selected={index === activeIndex}
              id={`${listId}-${index}`}
              key={item.objectId}
              className={index === activeIndex ? "active" : ""}
              onMouseDown={(event) => event.preventDefault()}
              onClick={() => select(item)}
            >
              {itemLabel(item)}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
