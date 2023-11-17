# ArgoCD Extensions

Forked from [argoproj-labs/argocd-extensions](https://github.com/argoproj-labs/argocd-extensions), commit [94e4126](https://github.com/argoproj-labs/argocd-extensions/tree/94e41261793f27c51de38f5c5544b1bb05a9e2e5).

## v0.1.0 (Base)

The ArgoCD Extensions operator was that it would watch for
`ArgoCDExtensions` CRs to be created, and it would then download all files within the `resources/`
directory. After an extension's resources files are downloaded, those files would be then moved into
a shared extensions directory.

The following sections detail the enhancements made to ArgoCD Extensions.

## v1.0.0 (Fork)

### Private GitHub Authentication

Adds support for authenticating with GitHub to access private repositories. 

The `ArgoCDExtension` CRD has been modified to allow for a Kubernetes secret to be provided by its name and namespace. 
A matching Kubernetes secret must be created with a valid SSH key as `data.sshkey`.

### Resource Customization ConfigMap

Adds resource customization ConfigMap in order to integrate with the ArgoCD server.

There was no existing functionality to integrate health checks, so a new ConfigMap has been introduced. This ConfigMap
is entirely owned by the ArgoCD Extensions operator and will be regenerated whenever the operator reconciles. After each
download of an ArgoCD Extension, the resource customizations will be read in from the file system and then placed into
this ConfigMap.

### Configurable Base Directory

The default base directory for a GitHub repo is `resources`. In order to be more flexible and allow for different 
repository structures, this can now be set in the `ArgoCDExtension` CR as `spec.baseDirectory`.

### Finalizer

Adds handling for the deletion of an `ArgoCDExtension` CR.

On deleting an `ArgoCDExtension` CR, the operator will delete all files that are part of the snapshot stored for the 
extension. Deletion will fail if the operator attempts to delete a file that isn't own by this extension. Once all files
are deleted, the ConfigMap is rebuilt and the finalizer is removed.

### Resource Conflicts

Adds handling for resource conflicts.

It is possible for two extensions to have the same resource file. Since all extensions are
eventually moved into a shared extensions folder, the order in which extensions are downloaded could
change the resulting files in the shared resources directory. If two extensions contain the same file,
then the resulting file would be from the most recently downloaded extension.

The Automation Platform fork of ArgoCD _prevents extensions from overwritting files owned by other extensions_.
This means that if two extensions contain the same file, then the first extension to be loaded will own the file.
The second extension will receive an error as it is not possible to load it without potentially breaking the existing
extension. 

Ownership is tracked using a file tracker. Whenever a file is downloaded, it is added to the file tracker. If the file
is already being tracked, then an error is thrown if the file isn't owned by current extension. If the file is owned by
current extension, then the file is overwritten and the file tracker is unchanged.

Preventing resource conflicts is critical to ensure that extensions owned by one team are not capable of
breaking extensions owned by another team.