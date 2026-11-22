fn main() {
    #[cfg(target_os = "macos")]
    {
        println!("cargo:rerun-if-changed=src-tauri/src/snapshot_mac.m");
        cc::Build::new()
            .file("src-tauri/src/snapshot_mac.m")
            .flag("-fobjc-arc")
            .compile("snapshot_mac");
        println!("cargo:rustc-link-lib=framework=Cocoa");
        println!("cargo:rustc-link-lib=framework=WebKit");
    }
    tauri_build::build();
}
