
// this file is generated — do not edit it


declare module "svelte/elements" {
	export interface HTMLAttributes<T> {
		'data-sveltekit-keepfocus'?: true | '' | 'off' | undefined | null;
		'data-sveltekit-noscroll'?: true | '' | 'off' | undefined | null;
		'data-sveltekit-preload-code'?:
			| true
			| ''
			| 'eager'
			| 'viewport'
			| 'hover'
			| 'tap'
			| 'off'
			| undefined
			| null;
		'data-sveltekit-preload-data'?: true | '' | 'hover' | 'tap' | 'off' | undefined | null;
		'data-sveltekit-reload'?: true | '' | 'off' | undefined | null;
		'data-sveltekit-replacestate'?: true | '' | 'off' | undefined | null;
	}
}

export {};


declare module "$app/types" {
	export interface AppTypes {
		RouteId(): "/" | "/business-units" | "/business-units/[id]" | "/technicians" | "/technicians/[id]";
		RouteParams(): {
			"/business-units/[id]": { id: string };
			"/technicians/[id]": { id: string }
		};
		LayoutParams(): {
			"/": { id?: string };
			"/business-units": { id?: string };
			"/business-units/[id]": { id: string };
			"/technicians": { id?: string };
			"/technicians/[id]": { id: string }
		};
		Pathname(): "/" | "/business-units" | "/business-units/" | `/business-units/${string}` & {} | `/business-units/${string}/` & {} | "/technicians" | "/technicians/" | `/technicians/${string}` & {} | `/technicians/${string}/` & {};
		ResolvedPathname(): `${"" | `/${string}`}${ReturnType<AppTypes['Pathname']>}`;
		Asset(): string & {};
	}
}